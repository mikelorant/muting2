package app

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mikelorant/muting2/internal/format"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gopkg.in/yaml.v3"
)

type Transforms struct {
	Transforms []Transform `yaml:"transforms"`
}

type Transform struct {
	From []string `yaml:"from"`
	To   string   `yaml:"to"`
}

func NewTransformer(ctx context.Context, f string) (*Transforms, error) {
	ctx, span := otel.Tracer(name).Start(ctx, "NewTransformer")
	defer span.End()

	var ts Transforms
	if err := load(ctx, f, &ts); err != nil {
		return nil, fmt.Errorf("unable to load transforms: %w", err)
	}

	return &ts, nil
}

func (ts *Transforms) Transform(ctx context.Context, str string) string {
	_, span := otel.Tracer(name).Start(ctx, "Transform")
	defer span.End()

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

func load(ctx context.Context, f string, v interface{}) error {
	_, span := otel.Tracer(name).Start(ctx, "load")
	defer span.End()

	fh, err := os.Open(f)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("unable to read transform file: %v: %w", f, err)
	}

	if err := yaml.NewDecoder(fh).Decode(v); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("unable to decode transform file: %v: %w", f, err)
	}

	return nil
}
