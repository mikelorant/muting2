package app

import (
	"fmt"

	"github.com/mikelorant/muting2/internal/tls"
)

type App struct {
	Options    Options
	Transforms *Transforms
	TLS        *tls.TLS
}

type Options struct {
	Bind      string
	Config    string
	Namespace string
	Name      string
	Service   string
}

func New(o Options) error {
	a := App{
		Options: o,
	}

	if err := a.DoTransformer(); err != nil {
		return fmt.Errorf("unable to do transformer: %w", err)
	}

	if err := a.DoTLS(); err != nil {
		return fmt.Errorf("unable to do TLS: %w", err)
	}

	if err := a.DoWebhookConfig(); err != nil {
		return fmt.Errorf("unable to do webhook: %w", err)
	}

	if err := a.DoServer(); err != nil {
		return fmt.Errorf("unable to do server: %w", err)
	}

	return nil
}

func (a *App) DoTransformer() error {
	t, err := NewTransformer(a.Options.Config)
	if err != nil {
		return fmt.Errorf("unable to do transformer: %w", err)
	}
	a.Transforms = t

	return nil
}

func (a *App) DoTLS() error {
	cn := fmt.Sprintf("%v.%v.svc", a.Options.Service, a.Options.Namespace)
	dn := []string{
		a.Options.Service,
		fmt.Sprintf("%v.%v", a.Options.Service, a.Options.Namespace),
		fmt.Sprintf("%v.%v.svc", a.Options.Service, a.Options.Namespace),
	}

	t, err := tls.NewTLS(tls.Options{
		CommonName: cn,
		DNSNames:   dn,
	})
	if err != nil {
		return fmt.Errorf("unable to get keypair: %w", err)
	}

	a.TLS = t

	return nil
}

func (a *App) DoWebhookConfig() error {
	wh := NewWebhookConfig(WebhookOptions{
		Name:      a.Options.Name,
		Namespace: a.Options.Namespace,
		Service:   a.Options.Service,
		CABundle:  a.TLS.CA.Get("certificate"),
	})

	if err := wh.Apply(); err != nil {
		return fmt.Errorf("unable to apply webhook: %w", err)
	}

	return nil
}

func (a *App) DoServer() error {
	h, err := NewWebhookHandler(a.Transforms)
	if err != nil {
		return fmt.Errorf("unable to get handler: %w", err)
	}

	if err := NewServer(ServerOptions{
		Addr:    a.Options.Bind,
		Handler: h,
		Keypair: a.TLS.Keypair,
	}); err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}

	return nil
}
