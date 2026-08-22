// Package static contains embedded browser assets.
package static

import "embed"

// FS contains all static assets.
//
//go:embed *
var FS embed.FS
