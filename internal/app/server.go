package app

import (
	cryptotls "crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mikelorant/muting2/internal/tls"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

func NewServer(o ServerOptions) error {
	s := Server{
		Options: o,
	}

	s.StartWithTLSKeypair(
		s.Options.Keypair.GetCertificate(),
		s.Options.Keypair.GetKey(),
	)

	return nil
}

func (s *Server) StartWithTLSKeypair(cert, key []byte) error {
	keypair, err := cryptotls.X509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("unable to assemble keypair: %w", err)
	}

	tlscfg := &cryptotls.Config{Certificates: []cryptotls.Certificate{keypair}}
	srv := &http.Server{
		Addr:         s.Options.Addr,
		TLSConfig:    tlscfg,
		Handler:      getRouter(s.Options.Webhook, s.Options.Metrics),
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}

	if err := srv.ListenAndServeTLS("", ""); err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}

	return nil
}

func getRouter(h Handler, m Metrics) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/status"))
	r.Mount("/debug", middleware.Profiler())
	r.Handle("/", h.Handler())
	r.Handle("/metrics", promhttp.InstrumentMetricHandler(m, promhttp.HandlerFor(m, promhttp.HandlerOpts{})))

	return r
}
