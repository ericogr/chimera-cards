package storage

import (
	"context"
	"time"

	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/imageutil"
	"github.com/ericogr/chimera-cards/internal/logging"
	"github.com/ericogr/chimera-cards/internal/openaiclient"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func OpenAndMigrate(dataSourceName string, entitiesFromConfig []game.Entity) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dataSourceName), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// In development, do not drop tables on startup anymore.
	// Keep schema updated via AutoMigrate and let 'make backend-clean' remove the DB when needed.
	err = db.AutoMigrate(&game.Entity{}, &game.Hybrid{}, &game.User{}, &game.Player{}, &game.Game{}, &game.HybridGeneratedName{})
	if err != nil {
		return nil, err
	}

	// Ensure a unique constraint across the three entity key columns. We use
	// an explicit UNIQUE index so combinations like (a,b,0) are enforced when
	// the third key is 0 (meaning "none"). The index targets the renamed
	// cache table for hybrid-generated assets.
	if execErr := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_hybrid_generated_cache_entities ON hybrid_generated_cache(entity1_key, entity2_key, entity3_key);").Error; execErr != nil {
		return nil, execErr
	}
	seedDefaultEntities(db, entitiesFromConfig)
	// Ensure entity images are present in the DB. If missing, generate via
	// OpenAI and resize to 256x256 before storing.
	ensureEntityImages(db, entitiesFromConfig)
	return db, nil
}

// ensureEntityImages checks for entity images and generates+stores any that
// are missing. This runs at startup and logs failures but does not abort
// startup on generation errors (so the server can still run offline).
func ensureEntityImages(db *gorm.DB, entitiesFromConfig []game.Entity) {
	// Fetch all entities from DB (excluding internal None).
	var entities []game.Entity
	if err := db.Where("name != ?", string(game.None)).Find(&entities).Error; err != nil {
		logging.Error("failed to fetch entities for image seeding", err, nil)
		return
	}

	for _, a := range entities {
		if len(a.ImagePNG) > 0 {
			// already present
			continue
		}
		// Generate using OpenAI (single-name prompt) and resize to 256x256.
		logging.Info("generating entity image", logging.Fields{"entity_id": a.ID, "name": a.Name})
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		imgBytes, err := openaiclient.GenerateEntityImage(ctx, a.Name)
		cancel()
		if err != nil {
			logging.Error("failed to generate image for entity", err, logging.Fields{"entity_id": a.ID, "name": a.Name})
			continue
		}
		resized, err := imageutil.ResizePNGBytes(imgBytes, 256, 256)
		if err != nil {
			logging.Error("failed to resize generated image", err, logging.Fields{"entity_id": a.ID, "name": a.Name})
			continue
		}
		if err := db.Model(&game.Entity{}).Where("id = ?", a.ID).Update("image_png", resized).Error; err != nil {
			logging.Error("failed to save generated image to DB", err, logging.Fields{"entity_id": a.ID, "name": a.Name})
			continue
		}
		logging.Info("entity image seeded", logging.Fields{"entity_id": a.ID, "name": a.Name})
	}
}

// (legacy DB migration removed) The entities' stats are always taken from
// `chimera_config.json` (config file). Any legacy-copy behavior was removed
// to keep the config the single source of truth.

func seedDefaultEntities(db *gorm.DB, entitiesFromConfig []game.Entity) {
	var count int64
	db.Model(&game.Entity{}).Count(&count)
	if count > 0 {
		return
	}
	// Build list to insert. Always include internal placeholder 'None'.
	entities := make([]game.Entity, 0, len(entitiesFromConfig)+1)
	// Insert the internal placeholder 'None' (other fields are provided by config and
	// are intentionally not persisted in the DB).
	entities = append(entities, game.Entity{Name: string(game.None)})
	// Append configured entities
	for _, a := range entitiesFromConfig {
		entities = append(entities, a)
	}
	db.Create(&entities)
}
