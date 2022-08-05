package app

import (
	"fmt"
	"log"
	"os"

	"github.com/common-nighthawk/go-figure"
	"github.com/mikelorant/muting2/internal/tls"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type App struct {
	Options    Options
	Transforms *Transforms
	TLS        *tls.TLS
	Registry   *prometheus.Registry
	Log        *log.Logger
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
		Options:  o,
		Registry: prometheus.NewRegistry(),
		Log:      log.New(os.Stdout, "", 0),
	}

	figure.NewFigure("Muting", "", true).Print()

	a.RegisterCollectors()

	if err := a.GetTransformer(); err != nil {
		return fmt.Errorf("unable to do transformer: %w", err)
	}

	if err := a.GetTLS(); err != nil {
		return fmt.Errorf("unable to do TLS: %w", err)
	}

	if err := a.ApplyAdmissionConfig(); err != nil {
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

	fmt.Println("Transforms:")
	fmt.Println(t)
	fmt.Println()

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

	fmt.Println("TLS Options:")
	fmt.Println(t.Options)
	fmt.Println()

	return nil
}

func (a *App) ApplyAdmissionConfig() error {
	ac := NewAdmissionConfig(AdmissionConfigOptions{
		Name:      a.Options.Name,
		Namespace: a.Options.Namespace,
		Service:   a.Options.Service,
		CABundle:  a.TLS.CA.GetCertificate(),
	})

	fmt.Println("Admission Config Options:")
	fmt.Println(ac.Options)
	fmt.Println()

	if err := ac.Apply(); err != nil {
		return fmt.Errorf("unable to apply webhook: %w", err)
	}

	a.Log.Println("Applied admission config.")

	return nil
}

func (a *App) StartServer() error {
	wh, err := NewWebhook(a.Transforms, a.Registry)
	if err != nil {
		return fmt.Errorf("unable to get handler: %w", err)
	}

	opts := ServerOptions{
		Addr:    a.Options.Bind,
		Webhook: wh,
		Metrics: a.Registry,
		Keypair: a.TLS.Keypair,
	}

	a.Log.Printf("Starting server [%v].\n", a.Options.Bind)

	if err := NewServer(opts); err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}

	return nil
}

func (a *App) RegisterCollectors() {
	a.Registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	a.Registry.MustRegister(collectors.NewGoCollector())
}
