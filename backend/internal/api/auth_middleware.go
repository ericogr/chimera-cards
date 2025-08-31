package api

import (
	"net/http"
	"os"
	"time"

	"github.com/ericogr/quimera-cards/internal/constants"
	"github.com/gin-gonic/gin"
)

// setSessionCookie sets the session cookie with appropriate flags for dev/prod.
func setSessionCookie(c *gin.Context, token string, ttl time.Duration) {
	secure := false
	if os.Getenv(constants.EnvSessionSecureCookie) == "1" {
		secure = true
	}
	c.SetCookie(constants.CookieSessionName, token, int(ttl.Seconds()), "/", "", secure, true)
}

func clearSessionCookie(c *gin.Context) {
	c.SetCookie(constants.CookieSessionName, "", -1, "/", "", false, true)
}

// AuthRequired validates the session cookie and injects identity into context.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(constants.CookieSessionName)
		if err != nil || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{constants.JSONKeyError: constants.ErrAuthRequired})
			return
		}
		claims, err := parseAndValidateSession(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{constants.JSONKeyError: constants.ErrInvalidSession})
			return
		}
		c.Set("userEmail", claims.Sub)
		c.Set("userName", claims.Name)
		c.Next()
	}
}
