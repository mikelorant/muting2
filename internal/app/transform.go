package app

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Transforms struct {
	Transforms []Transform `yaml:"transforms"`
}

type Transform struct {
	From []string `yaml:"from"`
	To   string   `yaml:"to"`
}

func NewTransformer(f string) (*Transforms, error) {
	var t Transforms
	if err := t.load(f); err != nil {
		return nil, fmt.Errorf("unable to load transforms: %w", err)
	}

	return &t, nil
}

func (ts *Transforms) load(f string) error {
	file, err := os.Open(f)
	if err != nil {
		return fmt.Errorf("unable to read transform file: %v: %w", f, err)
	}

	if err := yaml.NewDecoder(file).Decode(&ts); err != nil {
		return fmt.Errorf("unable to decode transform file: %v: %w", f, err)
	}

	return nil
}

func (ts *Transforms) Transform(str string) string {
	for _, r := range ts.Transforms {
		for _, suffix := range r.From {
			if !strings.HasSuffix(str, suffix) {
				continue
			}

			from := fmt.Sprintf(".%v$", suffix)
			to := fmt.Sprintf("$1.%v", r.To)

			re := regexp.MustCompile(from)
			result := re.ReplaceAllString(str, to)
			return result
		}
	}

	return str
}

func (r Transform) String() string {
	return fmt.Sprintf("Transform: From: %v To: %v\n", strings.Join(r.From, ", "), r.To)
}
