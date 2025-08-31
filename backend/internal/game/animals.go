package game

// AnimalName represents a canonical animal name used across the codebase.
// Using constants avoids typos and keeps references consistent.
type AnimalName string

const (
	// None represents an absent animal (used internally to mark "no third animal").
	// Its ID is represented by 0 in code and DB fields.
	None    AnimalName = "None"
	Lion    AnimalName = "Lion"
	Bear    AnimalName = "Bear"
	Cheetah AnimalName = "Cheetah"
	Eagle   AnimalName = "Eagle"
	Rhino   AnimalName = "Rhino"
	Turtle  AnimalName = "Turtle"
	Gorilla AnimalName = "Gorilla"
	Wolf    AnimalName = "Wolf"
	Octopus AnimalName = "Octopus"
	Raven   AnimalName = "Raven"
)
