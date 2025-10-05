package api

import (
	"encoding/json"
	"math/rand"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

const codeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const codeLength = 8

// generateJoinCode creates a short alphanumeric code for joining games.
func generateJoinCode() string {
	b := make([]byte, codeLength)
	for i := range b {
		b[i] = codeCharset[rand.Intn(len(codeCharset))]
	}
	return string(b)
}

var joinCodeRegex = regexp.MustCompile("^[A-Z0-9]{8}$")

func normalizeJoinCode(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// normalizeTimestamps recursively renames GORM timestamp keys from CamelCase
// (CreatedAt, UpdatedAt, DeletedAt) to snake_case keys (created_at, updated_at, deleted_at)
// so frontend clients consistently receive snake_case timestamps.
func normalizeTimestamps(v interface{}) interface{} {
	switch vv := v.(type) {
	case map[string]interface{}:
		for k, val := range vv {
			vv[k] = normalizeTimestamps(val)
		}
		if val, ok := vv["CreatedAt"]; ok {
			vv["created_at"] = val
			delete(vv, "CreatedAt")
		}
		if val, ok := vv["UpdatedAt"]; ok {
			vv["updated_at"] = val
			delete(vv, "UpdatedAt")
		}
		if val, ok := vv["DeletedAt"]; ok {
			vv["deleted_at"] = val
			delete(vv, "DeletedAt")
		}
		return vv
	case []interface{}:
		for i := range vv {
			vv[i] = normalizeTimestamps(vv[i])
		}
		return vv
	default:
		return v
	}
}

// MarshalIntoSnakeTimestamps marshals the given value into JSON, then decodes
// into an interface{} and normalizes timestamp keys to snake_case. It is used
// to produce API responses with consistent snake_case timestamp keys.
func MarshalIntoSnakeTimestamps(v interface{}) (interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return normalizeTimestamps(out), nil
}

// MarshalForContext behaves like MarshalIntoSnakeTimestamps but also
// redacts email fields that do not belong to the authenticated session
// user (if any). It inspects the gin.Context for a "userEmail" key and
// replaces any email-like values that don't match that value with an
// empty string so other users' emails are never exposed.
func MarshalForContext(c *gin.Context, v interface{}) (interface{}, error) {
	out, err := MarshalIntoSnakeTimestamps(v)
	if err != nil {
		return nil, err
	}
	currentEmail := ""
	if c != nil {
		if v, ok := c.Get("userEmail"); ok {
			if s, _ := v.(string); s != "" {
				currentEmail = s
			}
		}
	}
	redactEmails(out, currentEmail)
	return out, nil
}

// redactEmails walks a marshalled JSON structure (decoded into
// map[string]interface{} / []interface{}) and removes any field whose
// key contains "email" (case-insensitive) unless its value equals
// currentEmail. Removal deletes the key entirely so the JSON shape does
// not include email fields for other users.
func redactEmails(v interface{}, currentEmail string) {
	switch vv := v.(type) {
	case map[string]interface{}:
		for k, val := range vv {
			lower := strings.ToLower(k)
			if strings.Contains(lower, "email") {
				if s, ok := val.(string); ok {
					if currentEmail != "" && s == currentEmail {
						// keep the field when it matches the session user
						continue
					}
				}
				// Delete any email fields that don't belong to the session user
				delete(vv, k)
				continue
			}
			// Recurse into nested structures
			redactEmails(val, currentEmail)
		}
	case []interface{}:
		for i := range vv {
			redactEmails(vv[i], currentEmail)
		}
	default:
		// primitives: nothing to do
	}
}
