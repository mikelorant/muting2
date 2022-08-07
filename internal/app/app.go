package app

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/common-nighthawk/go-figure"
	"github.com/hackebrot/turtle"
	"github.com/mikelorant/muting2/internal/format"
	"github.com/mikelorant/muting2/internal/tls"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

type App struct {
	Options       Options
	Transforms    *Transforms
	TLS           *tls.TLS
	Observability Observability
	Log           *log.Logger
	Client        *kubernetes.Clientset
}

type Options struct {
	Bind      string
	Debug     bool
	Config    string
	Namespace string
	Name      string
	Service   string
}

const (
	name              = "github.com/mikelorant/muting2"
	tracerServiceName = "muting"
	profilerName      = "muting.app"
	profilerAddr      = "http://localhost:4040"
)

func New(o Options) error {
	a := App{
		Options: o,
		Log:     log.New(os.Stdout, "", 0),
	}

	figure.NewFigure("Muting", "", true).Print()

	ctx := context.Background()

	if err := a.ConfigureObservability(ctx); err != nil {
		return fmt.Errorf("unable to configure observability: %w", err)
	}
	defer a.Observability.TracerProvider.Shutdown(ctx)

	if err := a.GetTLS(ctx); err != nil {
		return fmt.Errorf("unable to do TLS: %w", err)
	}

	cl, err := newClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to get new client: %w", err)
	}
	a.Client = cl

	if err := a.GetTransformer(ctx); err != nil {
		return fmt.Errorf("unable to do transformer: %w", err)
	}

	if err := a.ApplyAdmissionConfig(ctx); err != nil {
		return fmt.Errorf("unable to do webhook: %w", err)
	}

	if err := a.StartServer(ctx); err != nil {
		return fmt.Errorf("unable to do server: %w", err)
	}

	return nil
}

func (a *App) ConfigureObservability(ctx context.Context) error {
	o := ObservabilityOptions{
		Debug:             a.Options.Debug,
		TracerServiceName: tracerServiceName,
		ProfilerName:      profilerName,
		ProfilerAddr:      profilerAddr,
	}

	obs, err := NewObservability(ctx, o)
	if err != nil {
		return fmt.Errorf("unable to setup observability: %w", err)
	}

	a.Observability = obs

	return nil
}

func (a *App) GetTransformer(ctx context.Context) error {
	ctx, span := otel.Tracer(name).Start(ctx, "GetTransformer")
	defer span.End()

	t, err := NewTransformer(TransformOptions{
		Namespace: a.Options.Namespace,
		Name:      a.Options.Name,
		Client:    a.Client.CoreV1(),
	})
	if err != nil {
		return fmt.Errorf("unable to create transformer: %w", err)
	}
	a.Transforms = t

	ts, err := t.Read(ctx)
	if err != nil {
		return fmt.Errorf("unable to read transforms: %w", err)
	}

	fmt.Println(turtle.Emojis["scissors"], " Transforms:")
	if len(ts) != 0 {
		fmt.Println(format.SliceToFormattedLines(ts))
	}
	fmt.Println()

	return nil
}

func (a *App) GetTLS(ctx context.Context) error {
	cn := fmt.Sprintf("%v.%v.svc", a.Options.Service, a.Options.Namespace)
	dn := []string{
		a.Options.Service,
		fmt.Sprintf("%v.%v", a.Options.Service, a.Options.Namespace),
		fmt.Sprintf("%v.%v.svc", a.Options.Service, a.Options.Namespace),
	}

	t, err := tls.NewTLS(ctx, tls.Options{
		CommonName: cn,
		DNSNames:   dn,
	})
	if err != nil {
		return fmt.Errorf("unable to get keypair: %w", err)
	}

	a.TLS = t

	fmt.Println(turtle.Emojis["lock"], "TLS Options:")
	fmt.Println(t.Options)
	fmt.Println()

	return nil
}

func (a *App) ApplyAdmissionConfig(ctx context.Context) error {
	ctx, span := otel.Tracer(name).Start(ctx, "ApplyAdmissionConfig")
	defer span.End()

	ac := NewAdmissionConfig(AdmissionConfigOptions{
		Client:    a.Client.AdmissionregistrationV1(),
		Name:      a.Options.Name,
		Namespace: a.Options.Namespace,
		Service:   a.Options.Service,
		CABundle:  a.TLS.CA.GetCertificate(),
	})

	fmt.Println(turtle.Emojis["vertical_traffic_light"], "Admission Config Options:")
	fmt.Println(ac.Options)
	fmt.Println()

	if err := ac.Apply(ctx); err != nil {
		return fmt.Errorf("unable to apply webhook: %w", err)
	}

	a.Log.Println(turtle.Emojis["art"], "Applied admission config.")

	return nil
}

func (a *App) StartServer(ctx context.Context) error {
	wh, err := NewWebhook(ctx, a.Transforms, a.Observability.Registry)
	if err != nil {
		return fmt.Errorf("unable to get handler: %w", err)
	}

	opts := ServerOptions{
		Addr:    a.Options.Bind,
		Webhook: wh,
		Metrics: a.Observability.Registry,
		Keypair: a.TLS.Keypair,
	}

	a.Log.Printf("%v Starting server [%v].\n", turtle.Emojis["white_check_mark"], a.Options.Bind)

	if err := NewServer(ctx, opts); err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}

	return nil
}

func newClient(ctx context.Context) (*kubernetes.Clientset, error) {
	_, span := otel.Tracer(name).Start(ctx, "newClient")
	defer span.End()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("unable to get config: %w", err)
	}

	cl, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("unable to create a new client: %w", err)
	}

	return cl, nil
}
