package app

import (
	cryptotls "crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/hellofresh/health-go/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mikelorant/muting2/internal/tls"
)

type Handler interface {
	Handler() http.Handler
}

type Server struct {
	Options ServerOptions
}

type ServerOptions struct {
	Keypair *tls.Keypair
	Addr    string
	Webhook Handler
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
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}

	hlth, err := health.New()
	if err != nil {
		return fmt.Errorf("unable to init health component: %w", err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Any("/*", echo.WrapHandler(s.Options.Webhook.Handler()))
	e.GET("/status", echo.WrapHandler(hlth.Handler()))

	if err := e.StartServer(srv); err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}

	return nil
}
