package storage

import (
	"context"
	"time"

	"github.com/ericogr/quimera-cards/internal/game"
	"github.com/ericogr/quimera-cards/internal/imageutil"
	"github.com/ericogr/quimera-cards/internal/logging"
	"github.com/ericogr/quimera-cards/internal/openaiclient"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func OpenAndMigrate(dataSourceName string, animalsFromConfig []game.Animal) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dataSourceName), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// In development, do not drop tables on startup anymore.
	// Keep schema updated via AutoMigrate and let 'make backend-clean' remove the DB when needed.
	err = db.AutoMigrate(&game.Animal{}, &game.Hybrid{}, &game.User{}, &game.Player{}, &game.Game{}, &game.HybridGeneratedName{})
	if err != nil {
		return nil, err
	}

	// Ensure a unique constraint across the three animal key columns. We use
	// an explicit UNIQUE index so combinations like (a,b,0) are enforced when
	// the third key is 0 (meaning "none"). The index targets the renamed
	// cache table for hybrid-generated assets.
	if execErr := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_hybrid_generated_cache_animals ON hybrid_generated_cache(animal1_key, animal2_key, animal3_key);").Error; execErr != nil {
		return nil, execErr
	}

	seedDefaultAnimals(db, animalsFromConfig)
	// Ensure animal images are present in the DB. If missing, generate via
	// OpenAI and resize to 256x256 before storing.
	ensureAnimalImages(db, animalsFromConfig)
	return db, nil
}

// ensureAnimalImages checks for animal images and generates+stores any that
// are missing. This runs at startup and logs failures but does not abort
// startup on generation errors (so the server can still run offline).
func ensureAnimalImages(db *gorm.DB, animalsFromConfig []game.Animal) {
	// Fetch all animals from DB (excluding internal None).
	var animals []game.Animal
	if err := db.Where("name != ?", string(game.None)).Find(&animals).Error; err != nil {
		logging.Error("failed to fetch animals for image seeding", err, nil)
		return
	}

	for _, a := range animals {
		if len(a.ImagePNG) > 0 {
			// already present
			continue
		}
		// Generate using OpenAI (single-name prompt) and resize to 256x256.
		logging.Info("generating animal image", logging.Fields{"animal_id": a.ID, "name": a.Name})
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		imgBytes, err := openaiclient.GenerateAnimalImage(ctx, a.Name)
		cancel()
		if err != nil {
			logging.Error("failed to generate image for animal", err, logging.Fields{"animal_id": a.ID, "name": a.Name})
			continue
		}
		resized, err := imageutil.ResizePNGBytes(imgBytes, 256, 256)
		if err != nil {
			logging.Error("failed to resize generated image", err, logging.Fields{"animal_id": a.ID, "name": a.Name})
			continue
		}
		if err := db.Model(&game.Animal{}).Where("id = ?", a.ID).Update("image_png", resized).Error; err != nil {
			logging.Error("failed to save generated image to DB", err, logging.Fields{"animal_id": a.ID, "name": a.Name})
			continue
		}
		logging.Info("animal image seeded", logging.Fields{"animal_id": a.ID, "name": a.Name})
	}
}

// (legacy DB migration removed) The animals' stats are always taken from
// `chimera_config.json` (config file). Any legacy-copy behavior was removed
// to keep the config the single source of truth.

func seedDefaultAnimals(db *gorm.DB, animalsFromConfig []game.Animal) {
	var count int64
	db.Model(&game.Animal{}).Count(&count)
	if count > 0 {
		return
	}
	// Build list to insert. Always include internal placeholder 'None'.
	animals := make([]game.Animal, 0, len(animalsFromConfig)+1)
	// Insert the internal placeholder 'None' (other fields are provided by config and
	// are intentionally not persisted in the DB).
	animals = append(animals, game.Animal{Name: string(game.None)})
	// Append configured animals
	for _, a := range animalsFromConfig {
		animals = append(animals, a)
	}
	db.Create(&animals)
}
