package hybridimage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ericogr/quimera-cards/internal/constants"
	"github.com/ericogr/quimera-cards/internal/dedupe"
	"github.com/ericogr/quimera-cards/internal/imageutil"
	"github.com/ericogr/quimera-cards/internal/keys"
	"github.com/ericogr/quimera-cards/internal/logging"
	"github.com/ericogr/quimera-cards/internal/openaiclient"
	"github.com/ericogr/quimera-cards/internal/storage"
)

// buildKeyFromNames produces the canonical entity key used to store hybrid
// images and generated names. It lower-cases, replaces spaces with
// underscores and sorts the parts so the key is order-independent.
// buildKeyFromNames removed; use keys.EntityKeyFromNames instead.

// EnsureHybridImage guarantees a hybrid image exists in the repository for
// the provided entity names. If the image is missing it will be generated
// via the OpenAI Images API, resized and saved. Concurrent requests for the
// same key are deduplicated using singleflight.
func EnsureHybridImage(repo storage.Repository, entityNames []string) error {
	if len(entityNames) == 0 {
		return fmt.Errorf("no entity names provided")
	}
	key := keys.EntityKeyFromNames(entityNames)

	// Fast path: already in DB
	if img, err := repo.GetHybridImageByKey(key); err == nil && len(img) > 0 {
		logging.Info("hybrid-image cache hit", logging.Fields{"entity_key": key, "size_bytes": len(img)})
		return nil
	}

	imgKey := "hybrid:" + key
	ch := dedupe.ImageGroup.DoChan(imgKey, func() (interface{}, error) {
		// Re-check repository in case another goroutine saved the image
		if img, err := repo.GetHybridImageByKey(key); err == nil && len(img) > 0 {
			return img, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		logging.Info("hybrid-image generating via OpenAI", logging.Fields{"entity_key": key, "entities": strings.Join(entityNames, " + ")})
		imgBytes, err := openaiclient.GenerateHybridImage(ctx, entityNames)
		if err != nil {
			return nil, err
		}
		out, err := imageutil.ResizePNGBytes(imgBytes, 256, 256)
		if err != nil {
			return nil, err
		}
		if err := repo.SaveHybridImageByKey(key, out); err != nil {
			logging.Error("failed to save hybrid image", err, logging.Fields{constants.LogFieldKey: key})
		} else {
			logging.Info("hybrid-image generated and saved", logging.Fields{"entity_key": key, "size_bytes": len(out)})
		}
		return out, nil
	})

	select {
	case r := <-ch:
		if r.Err != nil {
			logging.Error("hybrid image generation failed (singleflight)", r.Err, logging.Fields{constants.LogFieldKey: key})
			return r.Err
		}
		if _, ok := r.Val.([]byte); !ok {
			return fmt.Errorf("unexpected image result type")
		}
		return nil
	case <-time.After(90 * time.Second):
		logging.Error("hybrid image generation timed out", fmt.Errorf("timeout"), logging.Fields{constants.LogFieldKey: key})
		return fmt.Errorf("timed out waiting for image generation")
	}
}
