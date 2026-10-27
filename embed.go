// Package vulcanroot holds the files that the vulcan command embeds.
package vulcanroot

import _ "embed"

// Guide is the primer that vulcan guide prints (docs/poc/guide.md, spec §10).
//
//go:embed docs/poc/guide.md
var Guide string
