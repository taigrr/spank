package custom

import (
	"os"

	"github.com/taigrr/spank/internal/modepack"
)

func New(customPath string) *modepack.Pack {
	return &modepack.Pack{
		Name:   "custom",
		Source: os.DirFS(customPath),
		Dir:    ".",
		Mode:   modepack.ModeRandom,
	}
}

func NewFiles(files []string) *modepack.Pack {
	return &modepack.Pack{
		Name:   "custom",
		Mode:   modepack.ModeRandom,
		Custom: true,
		Files:  files,
	}
}
