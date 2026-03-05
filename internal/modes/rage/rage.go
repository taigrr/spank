package rage

import (
	"io/fs"

	"github.com/taigrr/spank/internal/modepack"
)

func New(audioFS fs.FS) *modepack.Pack {
	return &modepack.Pack{
		Name:          "rage",
		Source:        audioFS,
		Dir:           "audio/pain",
		Mode:          modepack.ModeEscalation,
		ShowRageMeter: true,
	}
}
