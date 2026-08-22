// Package templates contains the embedded HTML templates.
package templates

import "embed"

// FS contains all application templates.
//
//go:embed *.html pages/*.html
var FS embed.FS
