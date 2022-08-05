package app

import (
	"context"
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	utilpointer "k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
)

type AdmissionConfig struct {
	Config  *admissionregistrationv1.MutatingWebhookConfiguration
	Options AdmissionConfigOptions
}

type AdmissionConfigOptions struct {
	Namespace string
	Name      string
	Service   string
	CABundle  []byte
}

func NewAdmissionConfig(o AdmissionConfigOptions) AdmissionConfig {
	return AdmissionConfig{
		Config:  admissionConfig(o),
		Options: o,
	}
}

func (w *AdmissionConfig) Apply() error {
	cl, err := newClient()
	if err != nil {
		return fmt.Errorf("unable to create new client: %w", err)
	}

	ctx := context.TODO()
	mcl := cl.AdmissionregistrationV1().MutatingWebhookConfigurations()

	obj, err := mcl.Get(ctx, w.Options.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("unable to get admission config: %w", err)
	}

	if apierrors.IsNotFound(err) {
		if _, err := mcl.Create(ctx, w.Config, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("unable to create admission config: %w", err)
		}
		return nil
	}

	w.Config.ObjectMeta.ResourceVersion = obj.ObjectMeta.ResourceVersion
	if _, err := mcl.Update(ctx, w.Config, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("unable to update admission config: %w", err)
	}

	return nil
}

func newClient() (*kubernetes.Clientset, error) {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get config: %w", err)
	}

	cl, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create a new client: %w", err)
	}

	return cl, nil
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
