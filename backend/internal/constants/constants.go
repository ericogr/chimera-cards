package constants

// Centralized constants for headers, env keys and OpenAI integration.
const (
	// Environment variable keys
	EnvSessionSecret       = "SESSION_SECRET"
	EnvGoogleClientID      = "GOOGLE_CLIENT_ID"
	EnvGoogleClientSecret  = "GOOGLE_CLIENT_SECRET"
	EnvOpenAIAPIKey        = "OPENAI_API_KEY"
	EnvSessionSecureCookie = "SESSION_SECURE_COOKIE"

	// HTTP headers and content types
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"

	ContentTypeJSON = "application/json"
	ContentTypePNG  = "image/png"

	CacheControlHeader  = "Cache-Control"
	CacheControlNoCache = "no-cache, no-store, must-revalidate"

	// Authorization prefix
	BearerPrefix = "Bearer "

	// OpenAI API endpoints and base URL
	OpenAIBaseURL               = "https://api.openai.com"
	OpenAIChatCompletionsPath   = "/v1/chat/completions"
	OpenAIImagesGenerationsPath = "/v1/images/generations"
	OpenAIImagesEditsPath       = "/v1/images/edits"

	// OpenAI model names and typical parameters
	OpenAIChatModel           = "gpt-5-nano"
	OpenAIImageModel          = "gpt-image-1"
	OpenAIImageSizeDefault    = "1024x1024"
	OpenAIImageQualityDefault = "low"

	// OpenAI response JSON fields
	OpenAIResponseFieldB64JSON = "b64_json"

	// Session / Cookie names
	CookieSessionName = "q_session"

	// Google OAuth constants
	GoogleOAuthRedirect = "postmessage"
	GoogleUserInfoURL   = "https://www.googleapis.com/oauth2/v2/userinfo"
)

var (
	// Scopes for Google userinfo
	GoogleUserInfoScopes = []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"}
)

// Routes used by the backend router
const (
	RouteAPIPrefix          = "/api"
	RouteEntities           = "/entities"
	RouteEntitiesImage      = "/entities/image"
	RouteAssetsEntities     = "/assets/entities"
	RouteAssetsHybrids      = "/assets/hybrids"
	RoutePublicGames        = "/public-games"
	RouteLeaderboard        = "/leaderboard"
	RouteAuthGoogleCallBack = "/auth/google/oauth2callback"
	RoutePlayerStats        = "/player-stats"
	RouteGames              = "/games"
	RouteGamesJoin          = "/games/join"
	RouteGameByID           = "/games/:gameID"
	RouteGameStart          = "/games/:gameID/start"
	RouteGameEnd            = "/games/:gameID/end"
	RouteGameLeave          = "/games/:gameID/leave"
	RouteCreateHybrids      = "/games/:gameID/create-hybrids"
	RouteGameAction         = "/games/:gameID/action"
)

// Common JSON response keys
const (
	JSONKeyError   = "error"
	JSONKeyMessage = "message"
	JSONKeyDetails = "details"
	JSONKeyStatus  = "status"
	JSONKeyBody    = "body"
)

// Common error messages used across API handlers
const (
	ErrInvalidRequest         = "Invalid request"
	ErrMissingGoogleEnv       = "Missing GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET in environment"
	ErrInvalidGameID          = "Invalid game ID"
	ErrGameNotFound           = "Game not found"
	ErrFailedFetchEntities    = "Failed to fetch entities"
	ErrFailedFetchGames       = "Failed to fetch games"
	ErrFailedEncodeGames      = "Failed to encode games"
	ErrFailedFetchLeaderboard = "Failed to fetch leaderboard"
	ErrFailedEncodeGame       = "Failed to encode game"
	ErrFailedFetchStats       = "Failed to fetch stats"
	ErrEmailRequired          = "email is required"

	ErrFailedCreateGame             = "Failed to create game"
	ErrGameNameExceeds              = "Game name exceeds 32 characters"
	ErrDescriptionExceeds           = "Description exceeds 256 characters"
	ErrGameFull                     = "Game is full"
	ErrNotEnoughPlayers             = "Not enough players to start the game"
	ErrBothPlayersMustCreateHybrids = "Both players must create hybrids before starting"
	ErrGameAlreadyStartingOrStarted = "Game is already starting or started"
	ErrFailedUpdateGame             = "Failed to update game"
	ErrFailedUpdateGameStatus       = "Failed to update game status"
	ErrFailedEndGame                = "Failed to end game"
	ErrFailedRemovePlayer           = "Failed to remove player"
	ErrPlayerNotInThisGame          = "Player not in this game"
	ErrPlayerRemovedFailedUpdate    = "Player removed, but failed to update game"
	ErrCannotLeaveAfterGameStarted  = "Cannot leave after the game has started"

	ErrHybridsAlreadyCreated   = "Hybrids already created"
	ErrFailedSaveHybrids       = "Failed to save hybrids"
	ErrPlayerNotPartOfThisGame = "Player not part of this game"

	ErrFailedStoreAction           = "Failed to store action"
	ErrGameNotInProgress           = "Game is not in progress"
	ErrActionsLockedResolvingRound = "Actions are locked; resolving current round"
	ErrPlayerNotInGame             = "Player not in game"
	ErrNoActiveHybrid              = "No active hybrid"

	ErrFailedExchangeToken    = "Failed to exchange token"
	ErrFailedGetUserInfo      = "Failed to get user info"
	ErrFailedReadUserData     = "Failed to read user data: %s"
	ErrNoEmailInGoogleProfile = "No email in Google profile"
	ErrFailedCreateSession    = "Failed to create session"

	ErrAuthRequired   = "Authentication required"
	ErrInvalidSession = "Invalid session"
)

// animal_image specific errors and formats
const (
	ErrIDsParamRequired               = "ids query parameter is required (e.g. ids=1 or ids=1,2)"
	ErrIDsCountRange                  = "must provide between 1 and 3 entity ids"
	ErrInvalidIDFmt                   = "invalid id: %s"
	ErrEntitiesNotFoundFmt            = "entities not found: %s"
	ErrEnvNotSetFmt                   = "%s not set on server"
	ErrFailedCreateRequest            = "Failed to create request"
	ErrRequestToOpenAIFailed          = "Request to OpenAI failed"
	ErrOpenAIImageGenerationFailed    = "OpenAI image generation failed"
	ErrFailedDecodeOpenAIResponse     = "Failed to decode OpenAI response"
	ErrOpenAIReturnedNoImageData      = "OpenAI returned no image data"
	ErrFailedDecodeImageFromBase64    = "Failed to decode image from base64"
	ErrOpenAIReturnedUnsupportedImage = "OpenAI returned unsupported image payload"
	ErrFailedResizeImage              = "Failed to resize image"
)

// Logging field names
const (
	LogFieldGameID    = "game_id"
	LogFieldPlayerIdx = "player_index"
	LogFieldHybridIdx = "hybrid_index"
	LogFieldSource    = "source"
	LogFieldName      = "name"
	LogFieldKey       = "key"
	LogFieldAddr      = "addr"
)
