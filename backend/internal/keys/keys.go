package keys

import (
	"sort"
	"strings"
)

// EntityKeyFromNames produces a canonical key for a list of entity names.
// Behavior: trims names, lower-cases, replaces spaces with underscores,
// sorts the parts and joins with underscore. Suitable for stable DB keys.
func EntityKeyFromNames(names []string) string {
	parts := make([]string, 0, len(names))
	for _, n := range names {
		s := strings.TrimSpace(n)
		if s == "" {
			continue
		}
		s = strings.ToLower(strings.ReplaceAll(s, " ", "_"))
		parts = append(parts, s)
	}
	sort.Strings(parts)
	return strings.Join(parts, "_")
}
