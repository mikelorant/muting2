package tls

import (
	"errors"
	"fmt"
	"os"
	"path"
)

type PEMs struct {
	PEMStores map[string]PEMStore
}

type PEMStore struct {
	Filename string
	Data     []byte
}

var ErrPEMTypeUnknown = errors.New("unknown PEM type")

func (p *PEMs) WriteAll(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("unable to create directory: %w", err)
	}

	for _, v := range p.PEMStores {
		if err := os.WriteFile(path.Join(dir, v.Filename), v.Data, 0o644); err != nil {
			return fmt.Errorf("unable to create file: %v: %w", v.Filename, err)
		}
	}

	return nil
}

func (p *PEMs) Get(t string) []byte {
	switch t {
	case "key":
		return p.PEMStores["key"].Data
	case "certificate":
		return p.PEMStores["certificate"].Data
	default:
		return nil
	}
}
