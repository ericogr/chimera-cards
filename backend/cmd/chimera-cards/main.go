package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ericogr/chimera-cards/internal/api"
	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/ericogr/chimera-cards/internal/logging"

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
	cfg := loadConfigOrExit(configPath)
	applyPromptTemplates(cfg)

	// Allow the DB path to be configured via CHIMERA_DB. Default to
	// a `data/` directory inside the backend module for local development.
	dbPath := os.Getenv("CHIMERA_DB")
	if dbPath == "" {
		dbPath = "./data/chimera.db"
	}
	repo := createRepositoryOrExit(dbPath, cfg.Entities, cfg.PublicGamesTTL)
	handler := api.NewGameHandler(repo, cfg.ActionTimeout, cfg.PublicGamesTTL)

	// Worker identity for claim operations (unique per process start)
	workerID := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())

	startTimeoutScanner(repo, cfg.ActionTimeout, workerID)
	authHandler := api.NewAuthHandler(repo)

	// Create a fresh Gin engine and attach only the desired middleware.
	// Using `gin.New()` and explicitly adding `Logger`/`Recovery` avoids
	// a warning that occurs when the default middleware is attached multiple
	// times (for example in some environments or tests).
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	apiRoutes := router.Group(constants.RouteAPIPrefix)
	{
		// Public endpoints
		apiRoutes.GET(constants.RouteEntities, handler.ListEntities)
		apiRoutes.GET(constants.RoutePublicGames, handler.ListPublicGames)
		apiRoutes.GET(constants.RouteLeaderboard, handler.ListLeaderboard)
		apiRoutes.GET(constants.RouteConfig, handler.GetConfig)

		// Version information for debugging/releases
		apiRoutes.GET("/version", api.Version)

		// Authenticated endpoints
		protected := apiRoutes.Group("")
		protected.Use(api.AuthRequired())

		// Image and asset endpoints are protected — they require an authenticated session
		protected.GET(constants.RouteAssetsEntities+"/*file", handler.ServeEntityAsset)
		protected.GET(constants.RouteAssetsHybrids+"/*file", handler.ServeHybridAsset)

		protected.GET(constants.RoutePlayerStats, handler.GetPlayerStats)
		protected.POST(constants.RouteGames, handler.CreateGame)
		protected.POST(constants.RouteGamesJoin, handler.JoinGame)
		protected.GET(constants.RouteGameByCode, handler.GetGame)
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
