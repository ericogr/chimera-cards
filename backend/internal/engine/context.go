package engine

import (
	"strings"

	"github.com/ericogr/chimera-cards/internal/game"
)

// --- Round context and helpers ----------------------------------------
type roundContext struct {
	g       *game.Game
	summary []string
}

func newRoundContext(g *game.Game) *roundContext {
	return &roundContext{g: g, summary: make([]string, 0, 16)}
}

func (rc *roundContext) add(msg string) { rc.summary = append(rc.summary, msg) }

func (rc *roundContext) vulnerableTag(h *game.Hybrid) string {
	if h.VulnerableThisRound {
		return " — becomes Vulnerable (+25% damage this round)"
	}
	return ""
}

func (rc *roundContext) minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// joinSummary returns the accumulated summary as a single string.
func (rc *roundContext) joinSummary() string {
	return strings.Join(rc.summary, "\n")
}
