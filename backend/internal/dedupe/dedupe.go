package dedupe

// Package dedupe provides shared singleflight groups used to deduplicate
// concurrent generation requests (names and images). Using a centralized
// singleflight.Group ensures that only one generation job runs for a given
// key while other callers wait for the result.

import "golang.org/x/sync/singleflight"

// NameGroup deduplicates hybrid name generation requests keyed by the
// canonicalized list of animal IDs (e.g. "1,3,7").
var NameGroup singleflight.Group

// ImageGroup deduplicates image generation requests keyed by a unique
// string (for hybrids we use "hybrid:<key>", for animals "animal:<id>").
var ImageGroup singleflight.Group
