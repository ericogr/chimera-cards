package version

// These variables are overridden at build time using -ldflags.
// Keep sensible defaults for local development.
var (
	Version = "dev"
	Commit  = "none"
	Date    = ""
	Dirty   = "false"
)
