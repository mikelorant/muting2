package app

import (
	"context"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	networkingv1 "k8s.io/api/networking/v1"

	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
)

type Transformer interface {
	Transform(string) string
}

func NewWebhookHandler(t Transformer) (http.Handler, error) {
	whcfg := kwhmutating.WebhookConfig{
		ID:      "muting",
		Obj:     &networkingv1.Ingress{},
		Mutator: kwhmutating.MutatorFunc(mutatorFunc(t)),
	}

	wh, err := kwhmutating.NewWebhook(whcfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create webhook: %w", err)
	}

	whhcfg := kwhhttp.HandlerConfig{Webhook: wh}
	whh, err := kwhhttp.HandlerFor(whhcfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create webhook handler: %w", err)
	}

	return whh, nil
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
