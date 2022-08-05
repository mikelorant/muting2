package app

import (
	"fmt"
	"log"

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

	if err := a.GetTransformer(); err != nil {
		return fmt.Errorf("unable to do transformer: %w", err)
	}

	if err := a.GetTLS(); err != nil {
		return fmt.Errorf("unable to do TLS: %w", err)
	}

	if err := a.ApplyWebhookConfig(); err != nil {
		return fmt.Errorf("unable to do webhook: %w", err)
	}

	if err := a.StartServer(); err != nil {
		return fmt.Errorf("unable to do server: %w", err)
	}

	return nil
}

func (a *App) GetTransformer() error {
	t, err := NewTransformer(a.Options.Config)
	if err != nil {
		return fmt.Errorf("unable to do transformer: %w", err)
	}
	a.Transforms = t

	for _, v := range t.Transforms {
		log.Println(v)
	}

	return nil
}

func (a *App) GetTLS() error {
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

func (a *App) ApplyWebhookConfig() error {
	wh := NewWebhookConfig(WebhookConfigOptions{
		Name:      a.Options.Name,
		Namespace: a.Options.Namespace,
		Service:   a.Options.Service,
		CABundle:  a.TLS.CA.GetCertificate(),
	})

	if err := wh.Apply(); err != nil {
		return fmt.Errorf("unable to apply webhook: %w", err)
	}

	return nil
}

func (a *App) StartServer() error {
	webhook, err := NewWebhook(a.Transforms)
	if err != nil {
		return fmt.Errorf("unable to get handler: %w", err)
	}

	opts := ServerOptions{
		Addr:    a.Options.Bind,
		Webhook: webhook,
		Keypair: a.TLS.Keypair,
	}

	if err := NewServer(opts); err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}

	return nil
}
