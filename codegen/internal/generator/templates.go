package generator

import "embed"

// templatesFS embeds every Go text template used by the generator to render
// Java sources.
//
//go:embed templates/*.tmpl
var templatesFS embed.FS
