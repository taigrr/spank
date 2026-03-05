package modepack

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
)

type PlayMode int

const (
	ModeRandom PlayMode = iota
	ModeEscalation
)

type Pack struct {
	Name          string
	Source        fs.FS
	Dir           string
	Mode          PlayMode
	ShowRageMeter bool
	Custom        bool
	Files         []string
}

func (p *Pack) LoadFiles() error {
	if len(p.Files) > 0 {
		return nil
	}

	entries, err := fs.ReadDir(p.Source, p.Dir)
	if err != nil {
		return err
	}

	p.Files = make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		p.Files = append(p.Files, path.Join(p.Dir, entry.Name()))
	}
	sort.Strings(p.Files)

	if len(p.Files) == 0 {
		return fmt.Errorf("no audio files found in %s", p.Dir)
	}
	return nil
}

func (p *Pack) ReadFile(file string) ([]byte, error) {
	return fs.ReadFile(p.Source, file)
}
