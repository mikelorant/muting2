package app

import (
	"context"
	"fmt"
	"net/http"

	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/slok/kubewebhook/v2/pkg/webhook"
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

func NewWebhook(t Transformer) (*Webhook, error) {
	whcfg := kwhmutating.WebhookConfig{
		ID:      "muting",
		Obj:     &networkingv1.Ingress{},
		Mutator: kwhmutating.MutatorFunc(mutatorFunc(t)),
	}

	wh, err := kwhmutating.NewWebhook(whcfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create webhook: %w", err)
	}

	return &Webhook{
		Webhook: wh,
	}, nil
}

func (w *Webhook) Handler() http.Handler {
	whhcfg := kwhhttp.HandlerConfig{Webhook: w.Webhook}

	return kwhhttp.MustHandlerFor(whhcfg)
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
