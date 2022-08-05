package app

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mikelorant/muting2/internal/format"
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
	for _, t := range ts.Transforms {
		for _, suffix := range t.From {
			if !strings.HasSuffix(str, suffix) {
				continue
			}

			from := fmt.Sprintf(".%v$", suffix)
			to := fmt.Sprintf("$1.%v", t.To)

			re := regexp.MustCompile(from)
			return re.ReplaceAllString(str, to)
		}
	}

	return str
}

func (t Transform) String() string {
	return fmt.Sprintf("%v => %v", strings.Join(t.From, ", "), t.To)
}

func (ts Transforms) String() string {
	return format.SliceToFormattedLines(ts.Transforms)
}

func load(f string, v interface{}) error {
	fh, err := os.Open(f)
	if err != nil {
		return fmt.Errorf("unable to read transform file: %v: %w", f, err)
	}

	if err := yaml.NewDecoder(fh).Decode(v); err != nil {
		return fmt.Errorf("unable to decode transform file: %v: %w", f, err)
	}

	return nil
}
