package main

import (
	"os"
	"time"

	"github.com/ericogr/chimera-cards/internal/api"
	"github.com/ericogr/chimera-cards/internal/config"
	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/hybridname"
	"github.com/ericogr/chimera-cards/internal/logging"
	"github.com/ericogr/chimera-cards/internal/openaiclient"
	"github.com/ericogr/chimera-cards/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	checkEnvVars([]string{constants.EnvSessionSecret, constants.EnvGoogleClientID, constants.EnvGoogleClientSecret, constants.EnvOpenAIAPIKey})
	// Load entity configuration file (required). Path may be provided via
	// CHIMERA_CONFIG env var or defaults to ./chimera_config.json in the
	// current working directory.
	configPath := os.Getenv("CHIMERA_CONFIG")
	if configPath == "" {
		configPath = "./chimera_config.json"
	}
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logging.Fatal("Missing or invalid chimera configuration", err, logging.Fields{"config_path": configPath, "hint": "create a chimera_config.json with an 'entity_list' array of entity objects (name,hit_points,attack,defense,agility,energy,vigor_cost,skill{name,description,cost,key,effect}) and optional keys: server.address, single_image_prompt, hybrid_image_prompt"})
	}

	// If the configuration provides image prompt templates, apply them to
	// the OpenAI client so entity and hybrid image generation use the
	// configured texts.
	if cfg.SingleImagePromptTemplate != "" {
		openaiclient.SetSingleImagePromptTemplate(cfg.SingleImagePromptTemplate)
	}
	if cfg.HybridImagePromptTemplate != "" {
		openaiclient.SetHybridImagePromptTemplate(cfg.HybridImagePromptTemplate)
	}

	// If the configuration provides a name prompt template, apply it to
	// the hybrid name generator so name generation uses the configured text.
	if cfg.NamePromptTemplate != "" {
		hybridname.SetNamePromptTemplate(cfg.NamePromptTemplate)
	}

	// Allow the DB path to be configured via CHIMERA_DB. Default to
	// a `data/` directory inside the backend module for local development.
	dbPath := os.Getenv("CHIMERA_DB")
	if dbPath == "" {
		dbPath = "./data/chimera.db"
	}
	db, err := storage.OpenAndMigrate(dbPath, cfg.Entities)
	if err != nil {
		logging.Fatal("Failed to initialize database", err, nil)
	}

	repo := storage.NewSQLiteRepository(db, cfg.Entities, cfg.PublicGamesTTL)
	handler := api.NewGameHandler(repo, cfg.ActionTimeout)

	// Background scanner: periodically expire games whose action deadline
	// has passed. Expired games are finished with no winner and do not
	// affect player stats (StatsCounted=true prevents stat updates).
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			games, err := repo.FindTimedOutGames(now)
			if err != nil {
				logging.Error("timeout scanner failed", err, nil)
				continue
			}
			for _, g := range games {
				gg, err := repo.GetGameByID(g.ID)
				if err != nil {
					continue
				}
				if gg.Status != game.StatusInProgress || gg.Phase != game.PhasePlanning {
					continue
				}
				gg.Status = game.StatusFinished
				gg.Phase = game.PhaseResolved
				gg.Winner = ""
				gg.Message = "Match ended due to inactivity"
				// Build an English last-round summary describing which players
				// (if any) failed to submit actions before the deadline.
				summary := "Round timed out: "
				if len(gg.Players) == 2 {
					p1Submitted := gg.Players[0].HasSubmittedAction
					p2Submitted := gg.Players[1].HasSubmittedAction
					switch {
					case !p1Submitted && !p2Submitted:
						summary += "both players failed to submit actions within the allotted time."
					case p1Submitted && !p2Submitted:
						summary += gg.Players[1].PlayerName + " did not submit an action in time."
					case !p1Submitted && p2Submitted:
						summary += gg.Players[0].PlayerName + " did not submit an action in time."
					default:
						summary += "no resolution was reached."
					}
				} else {
					summary += "no resolution was reached due to inactivity."
				}
				gg.LastRoundSummary = summary
				gg.StatsCounted = true
				gg.ActionDeadline = time.Time{}
				if err := repo.UpdateGame(gg); err != nil {
					logging.Error("failed to expire game", err, logging.Fields{constants.LogFieldGameID: gg.ID})
				}
			}
		}
	}()
	authHandler := api.NewAuthHandler()

	router := gin.Default()

	apiRoutes := router.Group(constants.RouteAPIPrefix)
	{
		// Public endpoints
		apiRoutes.GET(constants.RouteEntities, handler.ListEntities)
		apiRoutes.GET(constants.RoutePublicGames, handler.ListPublicGames)
		apiRoutes.GET(constants.RouteLeaderboard, handler.ListLeaderboard)

		// Authenticated endpoints
		protected := apiRoutes.Group("")
		protected.Use(api.AuthRequired())

		// Image and asset endpoints are protected — they require an authenticated session
		protected.GET(constants.RouteAssetsEntities+"/*file", handler.ServeEntityAsset)
		protected.GET(constants.RouteAssetsHybrids+"/*file", handler.ServeHybridAsset)

		protected.GET(constants.RoutePlayerStats, handler.GetPlayerStats)
		protected.POST(constants.RouteGames, handler.CreateGame)
		protected.POST(constants.RouteGamesJoin, handler.JoinGame)
		protected.GET(constants.RouteGameByID, handler.GetGame)
		protected.POST(constants.RouteGameStart, handler.StartGame)
		protected.POST(constants.RouteGameEnd, handler.EndGame)
		protected.POST(constants.RouteGameLeave, handler.LeaveGame)
		protected.POST(constants.RouteCreateHybrids, handler.CreateHybrids)
		protected.POST(constants.RouteGameAction, handler.SubmitAction)
		// Player profile: GET returns stats, POST updates display name
		protected.POST(constants.RoutePlayerStats, handler.UpdatePlayerProfile)
	}

	router.POST(constants.RouteAuthGoogleCallBack, authHandler.GoogleOAuthCallback)

	// Start server on configured address
	addr := cfg.ServerAddress
	logging.Info("Server started", logging.Fields{constants.LogFieldAddr: addr})
	if err := router.Run(addr); err != nil {
		logging.Fatal("Failed to start server", err, nil)
	}
}

func checkEnvVars(vars []string) {
	for _, v := range vars {
		if os.Getenv(v) == "" {
			logging.Fatal("Required environment variable not set", nil, logging.Fields{"var": v})
		}
	}
}
