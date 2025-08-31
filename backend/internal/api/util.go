package api

import (
	"encoding/json"
	"math/rand"
)

const codeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const codeLength = 5

// generateJoinCode creates a short alphanumeric code for joining games.
func generateJoinCode() string {
	b := make([]byte, codeLength)
	for i := range b {
		b[i] = codeCharset[rand.Intn(len(codeCharset))]
	}
	return string(b)
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
