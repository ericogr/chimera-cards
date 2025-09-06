package game

// EntityName represents a canonical entity name used across the codebase.
// Using constants avoids typos and keeps references consistent.
type EntityName string

const (
	// None represents an absent entity (used internally to mark "no third entity").
	// Its ID is represented by 0 in code and DB fields.
	None EntityName = "None"
)
