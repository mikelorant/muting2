package tls

import (
	"fmt"
)

type TLS struct {
	CA      *CA
	Keypair *Keypair
}

type Options struct {
	CommonName string
	DNSNames   []string
}

func NewTLS(o Options) (*TLS, error) {
	ca, err := NewCA()
	if err != nil {
		return nil, fmt.Errorf("unable to create new CA: %w", err)
	}

	keypair, err := NewKeypair(KeypairOptions{
		CA:         ca,
		CommonName: o.CommonName,
		DNSNames:   o.DNSNames,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create new keypair: %w", err)
	}

	return &TLS{
		CA:      ca,
		Keypair: keypair,
	}, nil
}
