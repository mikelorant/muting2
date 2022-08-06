package app

import (
	"context"
	"fmt"

	"github.com/mikelorant/muting2/internal/format"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	admissionregistrationv1typed "k8s.io/client-go/kubernetes/typed/admissionregistration/v1"
	utilpointer "k8s.io/utils/pointer"
)

type AdmissionConfig struct {
	Client  admissionregistrationv1typed.MutatingWebhookConfigurationsGetter
	Config  *admissionregistrationv1.MutatingWebhookConfiguration
	Options AdmissionConfigOptions
}

type AdmissionConfigOptions struct {
	Namespace string
	Name      string
	Service   string
	CABundle  []byte
	Client    admissionregistrationv1typed.MutatingWebhookConfigurationsGetter
}

func NewAdmissionConfig(o AdmissionConfigOptions) AdmissionConfig {
	return AdmissionConfig{
		Client:  o.Client,
		Config:  admissionConfig(o),
		Options: o,
	}
}

func (w *AdmissionConfig) Apply(ctx context.Context) error {
	ctx, span := otel.Tracer(name).Start(ctx, "Apply")
	defer span.End()

	cl := w.Client.MutatingWebhookConfigurations()

	obj, err := cl.Get(ctx, w.Options.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("unable to get admission config: %w", err)
	}

	if apierrors.IsNotFound(err) {
		if _, err := cl.Create(ctx, w.Config, metav1.CreateOptions{}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("unable to create admission config: %w", err)
		}
		return nil
	}

	w.Config.ObjectMeta.ResourceVersion = obj.ObjectMeta.ResourceVersion
	if _, err := cl.Update(ctx, w.Config, metav1.UpdateOptions{}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("unable to update admission config: %w", err)
	}

	return nil
}

func admissionConfig(o AdmissionConfigOptions) *admissionregistrationv1.MutatingWebhookConfiguration {
	name := fmt.Sprintf("%v.%v.svc.cluster.local", o.Service, o.Namespace)
	fail := admissionregistrationv1.Fail
	sideEffect := admissionregistrationv1.SideEffectClassNone

	service := &admissionregistrationv1.ServiceReference{
		Name:      o.Name,
		Namespace: o.Namespace,
		Path:      utilpointer.String("/mutate"),
	}

	objectMeta := metav1.ObjectMeta{
		Name: o.Name,
	}

	clientConfig := admissionregistrationv1.WebhookClientConfig{
		CABundle: o.CABundle,
		Service:  service,
	}

	operations := []admissionregistrationv1.OperationType{
		admissionregistrationv1.Create,
		admissionregistrationv1.Update,
	}

	rule := admissionregistrationv1.Rule{
		APIGroups:   []string{"networking.k8s.io"},
		APIVersions: []string{"v1"},
		Resources:   []string{"ingresses"},
	}

	rules := []admissionregistrationv1.RuleWithOperations{{
		Operations: operations,
		Rule:       rule,
	}}

	matchLabels := map[string]string{
		(o.Service): "enabled",
	}

	namespaceSelector := &metav1.LabelSelector{
		MatchLabels: matchLabels,
	}

	webhooks := []admissionregistrationv1.MutatingWebhook{{
		Name:                    name,
		AdmissionReviewVersions: []string{"v1"},
		SideEffects:             &sideEffect,
		ClientConfig:            clientConfig,
		Rules:                   rules,
		NamespaceSelector:       namespaceSelector,
		FailurePolicy:           &fail,
	}}

	return &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: objectMeta,
		Webhooks:   webhooks,
	}
}

func (o AdmissionConfigOptions) String() string {
	return format.SliceToFormattedLines([]string{
		fmt.Sprintf("Namespace: %v", o.Namespace),
		fmt.Sprintf("Name: %v", o.Name),
		fmt.Sprintf("Service: %v", o.Service),
	})
}
