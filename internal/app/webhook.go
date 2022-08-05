package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhprometheus "github.com/slok/kubewebhook/v2/pkg/metrics/prometheus"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Transformer interface {
	Transform(string) string
}

type Webhook struct {
	Webhook webhook.Webhook
}

func NewWebhook(t Transformer, r prometheus.Registerer) (*Webhook, error) {
	whcfg := kwhmutating.WebhookConfig{
		ID:      "muting",
		Obj:     &networkingv1.Ingress{},
		Mutator: kwhmutating.MutatorFunc(mutatorFunc(t)),
	}

	wh, err := kwhmutating.NewWebhook(whcfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create webhook: %w", err)
	}

	rec, err := kwhprometheus.NewRecorder(kwhprometheus.RecorderConfig{Registry: r})
	if err != nil {
		return nil, fmt.Errorf("unable to create recorder: %w", err)
	}

	return &Webhook{
		Webhook: kwhwebhook.NewMeasuredWebhook(rec, wh),
	}, nil
}

func (w *Webhook) Handler() http.Handler {
	return kwhhttp.MustHandlerFor(kwhhttp.HandlerConfig{Webhook: w.Webhook})
}

func mutatorFunc(t Transformer) func(ctx context.Context, ar *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	return func(ctx context.Context, ar *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
		ing, ok := obj.(*networkingv1.Ingress)
		if !ok {
			return &kwhmutating.MutatorResult{}, nil
		}

		for idx, rule := range ing.Spec.Rules {
			rule.Host = t.Transform(rule.Host)
			ing.Spec.Rules[idx] = rule
		}

		return &kwhmutating.MutatorResult{MutatedObject: ing}, nil
	}
}
