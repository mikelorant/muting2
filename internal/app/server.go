package app

import (
	"context"
	cryptotls "crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mikelorant/muting2/internal/tls"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/sync/errgroup"
)

type Handler interface {
	Handler() http.Handler
}

type Metrics interface {
	prometheus.Registerer
	prometheus.Gatherer
}

type Server struct {
	Options ServerOptions
}

type ServerOptions struct {
	Keypair *tls.Keypair
	Addr    string
	Webhook Handler
	Metrics Metrics
}

func newServer(ctx context.Context, o ServerOptions) error {
	s := Server{
		Options: o,
	}

	s.startWithTLSKeypair(ctx,
		s.Options.Keypair.GetCertificate(),
		s.Options.Keypair.GetKey(),
	)

	return nil
}

func (s *Server) startWithTLSKeypair(ctx context.Context, cert, key []byte) error {
	keypair, err := cryptotls.X509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("unable to assemble keypair: %w", err)
	}

	tlscfg := &cryptotls.Config{Certificates: []cryptotls.Certificate{keypair}}
	srv := &http.Server{
		Addr:         s.Options.Addr,
		TLSConfig:    tlscfg,
		Handler:      getRouter(ctx, s.Options.Webhook, s.Options.Metrics),
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	g := new(errgroup.Group)

	g.Go(func() error {
		if err := srv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("unable to listen and serve: %w", err)
		}

		return nil
	})

	<-done

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("unable to shutdown server: %w", err)
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("go routine error: %w", err)
	}

	return nil
}

func getRouter(ctx context.Context, h Handler, m Metrics) *chi.Mux {
	wh := h.Handler()
	oh := otelhttp.NewHandler(wh, "Handler")
	ph := promhttp.InstrumentMetricHandler(m, promhttp.HandlerFor(m, promhttp.HandlerOpts{}))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/status"))
	r.Mount("/debug", middleware.Profiler())
	r.Handle("/", oh)
	r.Handle("/metrics", ph)

	return r
}
