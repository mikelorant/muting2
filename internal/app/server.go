package app

import (
	cryptotls "crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/mikelorant/muting2/internal/tls"
)

type Server struct {
	Options ServerOptions
}

type ServerOptions struct {
	Keypair *tls.Keypair
	Addr    string
	Handler http.Handler
}

func NewServer(o ServerOptions) error {
	s := Server{
		Options: o,
	}

	s.ListenAndServerTLSKeypair(
		s.Options.Keypair.Get("certificate"),
		s.Options.Keypair.Get("key"),
	)

	return nil
}

func (s *Server) ListenAndServerTLSKeypair(cert, key []byte) error {
	keypair, err := cryptotls.X509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("unable to assemble keypair: %w", err)
	}

	tlscfg := &cryptotls.Config{Certificates: []cryptotls.Certificate{keypair}}
	srv := &http.Server{
		Addr:         s.Options.Addr,
		Handler:      s.Options.Handler,
		TLSConfig:    tlscfg,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}

	if err := srv.ListenAndServeTLS("", ""); err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}

	return nil
}
