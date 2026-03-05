package halo

import (
	"io/fs"

	"github.com/taigrr/spank/internal/modepack"
)

func New(audioFS fs.FS) *modepack.Pack {
	return &modepack.Pack{
		Name:   "halo",
		Source: audioFS,
		Dir:    "audio/halo",
		Mode:   modepack.ModeRandom,
	}
}
