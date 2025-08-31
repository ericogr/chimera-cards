package hybridimage

import (
    "context"
    "fmt"
    "sort"
    "strings"
    "time"

    "github.com/ericogr/quimera-cards/internal/constants"
    "github.com/ericogr/quimera-cards/internal/dedupe"
    "github.com/ericogr/quimera-cards/internal/imageutil"
    "github.com/ericogr/quimera-cards/internal/logging"
    "github.com/ericogr/quimera-cards/internal/openaiclient"
    "github.com/ericogr/quimera-cards/internal/storage"
)

// buildKeyFromNames produces the canonical animal key used to store hybrid
// images and generated names. It lower-cases, replaces spaces with
// underscores and sorts the parts so the key is order-independent.
func buildKeyFromNames(names []string) string {
    cleaned := make([]string, 0, len(names))
    for _, n := range names {
        q := strings.TrimSpace(n)
        if q == "" {
            continue
        }
        cleaned = append(cleaned, strings.ToLower(strings.ReplaceAll(q, " ", "_")))
    }
    sort.Strings(cleaned)
    return strings.Join(cleaned, "_")
}

// EnsureHybridImage guarantees a hybrid image exists in the repository for
// the provided animal names. If the image is missing it will be generated
// via the OpenAI Images API, resized and saved. Concurrent requests for the
// same key are deduplicated using singleflight.
func EnsureHybridImage(repo storage.Repository, animalNames []string) error {
    if len(animalNames) == 0 {
        return fmt.Errorf("no animal names provided")
    }
    key := buildKeyFromNames(animalNames)

    // Fast path: already in DB
    if img, err := repo.GetHybridImageByKey(key); err == nil && len(img) > 0 {
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

        imgBytes, err := openaiclient.GenerateImageFromNames(ctx, animalNames)
        if err != nil {
            return nil, err
        }
        out, err := imageutil.ResizePNGBytes(imgBytes, 256, 256)
        if err != nil {
            return nil, err
        }
        if err := repo.SaveHybridImageByKey(key, out); err != nil {
            logging.Error("failed to save hybrid image", err, logging.Fields{constants.LogFieldKey: key})
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

