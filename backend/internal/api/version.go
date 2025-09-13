package api

import (
	"net/http"

	"github.com/ericogr/chimera-cards/internal/version"
	"github.com/gin-gonic/gin"
)

// Version returns build and VCS metadata injected at build time.
func Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": version.Version,
		"commit":  version.Commit,
		"date":    version.Date,
		"dirty":   version.Dirty,
	})
}
