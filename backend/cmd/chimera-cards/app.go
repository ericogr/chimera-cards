package main

import (
	"time"

	"github.com/ericogr/chimera-cards/internal/config"
	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/hybridname"
	"github.com/ericogr/chimera-cards/internal/logging"
	"github.com/ericogr/chimera-cards/internal/openaiclient"
	"github.com/ericogr/chimera-cards/internal/storage"
)

func loadConfigOrExit(path string) *config.LoadedConfig {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		logging.Fatal("Missing or invalid chimera configuration", err, logging.Fields{"config_path": path})
	}
	return cfg
}

func applyPromptTemplates(cfg *config.LoadedConfig) {
	if cfg == nil {
		return
	}
	if cfg.SingleImagePromptTemplate != "" {
		openaiclient.SetSingleImagePromptTemplate(cfg.SingleImagePromptTemplate)
	}
	if cfg.HybridImagePromptTemplate != "" {
		openaiclient.SetHybridImagePromptTemplate(cfg.HybridImagePromptTemplate)
	}
	if cfg.NamePromptTemplate != "" {
		hybridname.SetNamePromptTemplate(cfg.NamePromptTemplate)
	}
}

func createRepositoryOrExit(dbPath string, entities []game.Entity, publicGamesTTL time.Duration) storage.Repository {
	db, err := storage.OpenDB(dbPath, entities)
	if err != nil {
		logging.Fatal("Failed to initialize database", err, nil)
	}
	return storage.NewSQLiteRepository(db, entities, publicGamesTTL)
}
