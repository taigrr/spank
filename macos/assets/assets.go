// Package assets embeds the static assets for the SpankUI app.
package assets

import _ "embed"

//go:embed icon_template.png
var IconTemplate []byte

//go:embed icon_regular.png
var IconRegular []byte
