package game

// AnimalName represents a canonical animal name used across the codebase.
// Using constants avoids typos and keeps references consistent.
type AnimalName string

const (
	// None represents an absent animal (used internally to mark "no third animal").
	// Its ID is represented by 0 in code and DB fields.
	None AnimalName = "None"
)
