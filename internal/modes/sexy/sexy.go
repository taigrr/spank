package sexy

import (
	"io/fs"

	"github.com/taigrr/spank/internal/modepack"
)

func New(audioFS fs.FS) *modepack.Pack {
	return &modepack.Pack{
		Name:   "sexy",
		Source: audioFS,
		Dir:    "audio/sexy",
		Mode:   modepack.ModeEscalation,
	}
}
