package app

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type Transforms struct {
	Client  corev1.ConfigMapsGetter
	Options TransformOptions
}

type Transform struct {
	From []string `yaml:"from"`
	To   string   `yaml:"to"`
}

type TransformOptions struct {
	Namespace string
	Name      string
	Client    corev1.ConfigMapsGetter
}

func NewTransformer(o TransformOptions) (*Transforms, error) {
	ts := Transforms{
		Options: o,
		Client:  o.Client,
	}

	return &ts, nil
}

func (ts *Transforms) Transform(ctx context.Context, str string) (string, error) {
	ctx, span := otel.Tracer(name).Start(ctx, "Transform")
	defer span.End()

	tt, err := ts.Read(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return str, fmt.Errorf("unable to read config map: %w", err)
	}

	for _, t := range tt {
		for _, suffix := range t.From {
			if !strings.HasSuffix(str, suffix) {
				continue
			}

			from := fmt.Sprintf(".%v$", suffix)
			to := fmt.Sprintf("$1.%v", t.To)

			re := regexp.MustCompile(from)
			return re.ReplaceAllString(str, to), nil
		}
	}

	return str, nil
}

func (t Transform) String() string {
	return fmt.Sprintf("%v => %v", strings.Join(t.From, ", "), t.To)
}

func (ts *Transforms) Read(ctx context.Context) ([]Transform, error) {
	ctx, span := otel.Tracer(name).Start(ctx, "read")
	defer span.End()

	var tt []Transform

	cl := ts.Client.ConfigMaps(ts.Options.Namespace)

	cm, err := cl.Get(ctx, ts.Options.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return tt, fmt.Errorf("unable to get config map: %v: %w", ts.Options.Name, err)
	}

	data, ok := cm.Data["transforms"]
	if !ok {
		return tt, nil
	}

	if err := yaml.Unmarshal([]byte(data), &tt); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return tt, fmt.Errorf("unable to unmarshal config map: %w", err)
	}

	return tt, nil
}
