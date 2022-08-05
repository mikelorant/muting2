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
	var ts Transforms
	if err := load(f, &ts); err != nil {
		return nil, fmt.Errorf("unable to load transforms: %w", err)
	}

	return &ts, nil
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

func (t Transform) String() string {
	return fmt.Sprintf("Transform: From: %v To: %v", strings.Join(t.From, ", "), t.To)
}

func load(f string, v interface{}) error {
	file, err := os.Open(f)
	if err != nil {
		return fmt.Errorf("unable to read transform file: %v: %w", f, err)
	}

	if err := yaml.NewDecoder(file).Decode(v); err != nil {
		return fmt.Errorf("unable to decode transform file: %v: %w", f, err)
	}

	return nil
}
