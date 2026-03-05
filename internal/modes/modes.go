package modes

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/taigrr/spank/internal/modepack"
	"github.com/taigrr/spank/internal/modes/custom"
	"github.com/taigrr/spank/internal/modes/halo"
	"github.com/taigrr/spank/internal/modes/pain"
	"github.com/taigrr/spank/internal/modes/rage"
	"github.com/taigrr/spank/internal/modes/sexy"
)

type Selection struct {
	Sexy        bool
	Halo        bool
	Rage        bool
	CustomPath  string
	CustomFiles []string
}

func Resolve(sel Selection, painFS fs.FS, sexyFS fs.FS, haloFS fs.FS) (*modepack.Pack, error) {
	modeCount := 0
	if sel.Sexy {
		modeCount++
	}
	if sel.Halo {
		modeCount++
	}
	if sel.Rage {
		modeCount++
	}
	if sel.CustomPath != "" || len(sel.CustomFiles) > 0 {
		modeCount++
	}
	if modeCount > 1 {
		return nil, fmt.Errorf("--sexy, --halo, --rage, and --custom/--custom-files are mutually exclusive; pick one")
	}

	if len(sel.CustomFiles) > 0 {
		for _, file := range sel.CustomFiles {
			if !strings.HasSuffix(strings.ToLower(file), ".mp3") {
				return nil, fmt.Errorf("custom file must be MP3: %s", file)
			}
			if _, err := os.Stat(file); err != nil {
				return nil, fmt.Errorf("custom file not found: %s", file)
			}
		}
		return custom.NewFiles(sel.CustomFiles), nil
	}

	switch {
	case sel.CustomPath != "":
		return custom.New(sel.CustomPath), nil
	case sel.Rage:
		return rage.New(painFS), nil
	case sel.Sexy:
		return sexy.New(sexyFS), nil
	case sel.Halo:
		return halo.New(haloFS), nil
	default:
		return pain.New(painFS), nil
	}
}
