package main

import (
	"os"

	"github.com/ericogr/quimera-cards/internal/api"
	"github.com/ericogr/quimera-cards/internal/config"
	"github.com/ericogr/quimera-cards/internal/constants"
	"github.com/ericogr/quimera-cards/internal/hybridname"
	"github.com/ericogr/quimera-cards/internal/logging"
	"github.com/ericogr/quimera-cards/internal/openaiclient"
	"github.com/ericogr/quimera-cards/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	checkEnvVars([]string{constants.EnvSessionSecret, constants.EnvGoogleClientID, constants.EnvGoogleClientSecret, constants.EnvOpenAIAPIKey})
	// Load animal configuration file (required). Path may be provided via
	// CHIMERA_CONFIG env var or defaults to ./chimera_config.json in the
	// current working directory.
	configPath := os.Getenv("CHIMERA_CONFIG")
	if configPath == "" {
		configPath = "./chimera_config.json"
	}
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logging.Fatal("Missing or invalid chimera configuration", err, logging.Fields{"config_path": configPath, "hint": "create a chimera_config.json with an 'animal_list' array of animal objects (name,hit_points,attack,defense,agility,energy,skill_name,skill_cost,skill_description) and optional server.address"})
	}

	// If the configuration provides an image prompt template, apply it
	// to the OpenAI client so image generation uses the configured text.
	if cfg.ImagePromptTemplate != "" {
		openaiclient.SetImagePromptTemplate(cfg.ImagePromptTemplate)
	}

	// If the configuration provides a name prompt template, apply it to
	// the hybrid name generator so name generation uses the configured text.
	if cfg.NamePromptTemplate != "" {
		hybridname.SetNamePromptTemplate(cfg.NamePromptTemplate)
	}

	db, err := storage.OpenAndMigrate("quimera.db", cfg.Animals)
	if err != nil {
		logging.Fatal("Falha ao inicializar o banco de dados", err, nil)
	}

	repo := storage.NewSQLiteRepository(db, cfg.Animals)
	handler := api.NewGameHandler(repo)
	authHandler := api.NewAuthHandler()

	router := gin.Default()

	apiRoutes := router.Group(constants.RouteAPIPrefix)
	{
		// Public endpoints
		apiRoutes.GET(constants.RouteAnimals, handler.ListAnimals)
		apiRoutes.GET(constants.RoutePublicGames, handler.ListPublicGames)
		apiRoutes.GET(constants.RouteLeaderboard, handler.ListLeaderboard)

		authRoutes := apiRoutes.Group(constants.RouteAuth)
		{
			authRoutes.POST("/google/oauth2callback", authHandler.GoogleOAuthCallback)
		}

		// Authenticated endpoints
		protected := apiRoutes.Group("")
		protected.Use(api.AuthRequired())

		// Image and asset endpoints are protected — they require an authenticated session
		protected.GET(constants.RouteAnimalsImage, handler.GenerateAnimalImage)
		protected.GET(constants.RouteAssetsAnimals+"/*file", handler.ServeAnimalAsset)
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
	}

	// Start server on configured address
	addr := cfg.ServerAddress
	// For logging present a http://localhost:PORT style when address starts with ':'
	displayAddr := addr
	if len(addr) > 0 && addr[0] == ':' {
		displayAddr = "http://localhost" + addr
	}
	logging.Info("Servidor iniciado", logging.Fields{constants.LogFieldAddr: displayAddr})
	if err := router.Run(addr); err != nil {
		logging.Fatal("Falha ao iniciar o servidor", err, nil)
	}
}

func checkEnvVars(vars []string) {
	for _, v := range vars {
		if os.Getenv(v) == "" {
			logging.Fatal("Variável de ambiente obrigatória não definida", nil, logging.Fields{"var": v})
		}
	}
}
