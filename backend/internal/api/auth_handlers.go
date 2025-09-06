package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

type GoogleOAuthCallbackRequest struct {
	Code string `json:"code"`
}

func (h *AuthHandler) GoogleOAuthCallback(c *gin.Context) {
	var req GoogleOAuthCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidRequest})
		return
	}

	googleClientID := os.Getenv(constants.EnvGoogleClientID)
	googleClientSecret := os.Getenv(constants.EnvGoogleClientSecret)
	if googleClientID == "" || googleClientSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrMissingGoogleEnv})
		return
	}

	conf := &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		RedirectURL:  constants.GoogleOAuthRedirect,
		Scopes:       constants.GoogleUserInfoScopes,
		Endpoint:     google.Endpoint,
	}

	token, err := conf.Exchange(context.Background(), req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{constants.JSONKeyError: constants.ErrFailedExchangeToken, constants.JSONKeyDetails: err.Error()})
		return
	}

	client := conf.Client(context.Background(), token)
	resp, err := client.Get(constants.GoogleUserInfoURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedGetUserInfo, constants.JSONKeyDetails: err.Error()})
		return
	}
	defer resp.Body.Close()

	userData, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: fmt.Sprintf(constants.ErrFailedReadUserData, err.Error())})
		return
	}

	// Parse minimal fields from user info
	var payload map[string]any
	_ = json.Unmarshal(userData, &payload)
	email, _ := payload["email"].(string)
	name, _ := payload["name"].(string)
	if email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{constants.JSONKeyError: constants.ErrNoEmailInGoogleProfile})
		return
	}

	// Mint session token and set cookie
	sess, err := createSessionToken(email, name, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedCreateSession, constants.JSONKeyDetails: err.Error()})
		return
	}
	setSessionCookie(c, sess, 24*time.Hour)

	// Upsert user profile (best-effort) via repository if available
	// We don't have repo here; login is independent. The game routes will upsert on usage.
	c.Data(http.StatusOK, constants.ContentTypeJSON, userData)
}
