package templates

import "embed"

//go:embed *.tmpl
var FS embed.FS

// ValidStyles lists the available STYLE.md variants, keyed by flag value.
var ValidStyles = []string{"first-person", "third-close", "third-omniscient"}
