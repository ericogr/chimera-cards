package api

import (
	"context"
	"net/http"
	"path"
	"strings"
	"time"

	"fmt"
	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/ericogr/chimera-cards/internal/dedupe"
	"github.com/ericogr/chimera-cards/internal/imageutil"
	"github.com/ericogr/chimera-cards/internal/logging"
	"github.com/ericogr/chimera-cards/internal/openaiclient"
	"github.com/gin-gonic/gin"
)

// ServeEntityAsset serves entity images stored in the DB. URL format:
// /api/assets/entities/<name>.png
func (h *GameHandler) ServeEntityAsset(c *gin.Context) {
	file := c.Param("file") // includes leading '/'
	if strings.HasPrefix(file, "/") {
		file = file[1:]
	}
	if file == "" {
		c.Status(http.StatusNotFound)
		return
	}

	name := strings.TrimSuffix(file, path.Ext(file))
	// Lookup entity by name (case-insensitive)
	a, err := h.repo.GetEntityByName(name)
	if err != nil || a == nil {
		c.Status(http.StatusNotFound)
		return
	}

	if len(a.ImagePNG) > 0 {
		c.Header(constants.HeaderContentType, constants.ContentTypePNG)
		c.Header(constants.CacheControlHeader, constants.CacheControlNoCache)
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.Write(a.ImagePNG)
		return
	}

	// Not found in DB — deduplicate concurrent generation using singleflight.
	logging.Info("entity image not found in DB; generating (or joining existing)", logging.Fields{"name": a.Name})
	key := fmt.Sprintf("entity:%d", a.ID)
	ch := dedupe.ImageGroup.DoChan(key, func() (interface{}, error) {
		// Re-check DB in case another goroutine saved it while we were queued.
		if a2, err := h.repo.GetEntityByName(a.Name); err == nil && a2 != nil && len(a2.ImagePNG) > 0 {
			return a2.ImagePNG, nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		imgBytes, err := openaiclient.GenerateEntityImage(ctx, a.Name)
		if err != nil {
			return nil, err
		}
		out, err := imageutil.ResizePNGBytes(imgBytes, 256, 256)
		if err != nil {
			return nil, err
		}
		if err := h.repo.SaveEntityImage(a.ID, out); err != nil {
			logging.Error("failed to save generated entity image", err, logging.Fields{"entity_id": a.ID})
		}
		return out, nil
	})

	select {
	case r := <-ch:
		if r.Err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrOpenAIImageGenerationFailed, constants.JSONKeyDetails: r.Err.Error()})
			return
		}
		out, ok := r.Val.([]byte)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrOpenAIImageGenerationFailed, constants.JSONKeyDetails: "invalid image result"})
			return
		}
		c.Header(constants.HeaderContentType, constants.ContentTypePNG)
		c.Header(constants.CacheControlHeader, constants.CacheControlNoCache)
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.Write(out)
		return
	case <-time.After(90 * time.Second):
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrOpenAIImageGenerationFailed, constants.JSONKeyDetails: "image generation timed out"})
		return
	}
}

// ServeHybridAsset serves or generates a hybrid image identified by a key
// formed by joining entity names (sorted alphabetically, lowercase) with
// underscores. URL format: /api/assets/hybrids/<name1>_<name2>.png
func (h *GameHandler) ServeHybridAsset(c *gin.Context) {
	file := c.Param("file")
	if strings.HasPrefix(file, "/") {
		file = file[1:]
	}
	if file == "" {
		c.Status(http.StatusNotFound)
		return
	}
	key := strings.TrimSuffix(file, path.Ext(file))
	key = strings.ToLower(key)

	// Try DB
	if img, err := h.repo.GetHybridImageByKey(key); err == nil && len(img) > 0 {
		c.Header(constants.HeaderContentType, constants.ContentTypePNG)
		c.Header(constants.CacheControlHeader, constants.CacheControlNoCache)
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.Write(img)
		return
	}

	// Not found: reconstruct names from key and map to canonical entity names
	parts := strings.Split(key, "_")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if a, err := h.repo.GetEntityByName(p); err == nil && a != nil {
			names = append(names, a.Name)
		} else {
			// Fallback: replace underscores with spaces and title-case
			s := strings.ReplaceAll(p, "_", " ")
			names = append(names, strings.Title(s))
		}
	}

	if len(names) == 0 {
		c.Status(http.StatusNotFound)
		return
	}

	// Generate via OpenAI and resize — deduplicate concurrent requests using
	// singleflight so only the first caller performs the heavy work.
	logging.Info("generating hybrid image (or joining existing)", logging.Fields{"key": key, "names": strings.Join(names, ",")})
	imgKey := fmt.Sprintf("hybrid:%s", key)
	ch := dedupe.ImageGroup.DoChan(imgKey, func() (interface{}, error) {
		// Re-check the DB in case another goroutine saved it while queued.
		if img, err := h.repo.GetHybridImageByKey(key); err == nil && len(img) > 0 {
			return img, nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		imgBytes, err := openaiclient.GenerateHybridImage(ctx, names)
		if err != nil {
			return nil, err
		}
		out, err := imageutil.ResizePNGBytes(imgBytes, 256, 256)
		if err != nil {
			return nil, err
		}
		if err := h.repo.SaveHybridImageByKey(key, out); err != nil {
			logging.Error("failed to save hybrid image", err, logging.Fields{"key": key})
		}
		return out, nil
	})

	select {
	case r := <-ch:
		if r.Err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrOpenAIImageGenerationFailed, constants.JSONKeyDetails: r.Err.Error()})
			return
		}
		out, ok := r.Val.([]byte)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrOpenAIImageGenerationFailed, constants.JSONKeyDetails: "invalid image result"})
			return
		}
		c.Header(constants.HeaderContentType, constants.ContentTypePNG)
		c.Header(constants.CacheControlHeader, constants.CacheControlNoCache)
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.Write(out)
		return
	case <-time.After(90 * time.Second):
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrOpenAIImageGenerationFailed, constants.JSONKeyDetails: "image generation timed out"})
		return
	}
}
