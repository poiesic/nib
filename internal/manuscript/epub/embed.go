package epub

import _ "embed"

//go:embed epub.css
var CSS []byte

//go:embed fonts/Literata.ttf
var FontRegular []byte

//go:embed fonts/Literata-Italic.ttf
var FontItalic []byte
