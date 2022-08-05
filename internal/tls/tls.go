package tls

import (
	"fmt"
	"strings"

	"github.com/mikelorant/muting2/internal/format"
)

type TLS struct {
	CA      *CA
	Keypair *Keypair
	Options Options
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
		Options: o,
	}, nil
}

func (o Options) String() string {
	var sb strings.Builder
	fmt.Fprintln(&sb, "Common Name:", o.CommonName)
	fmt.Fprint(&sb, format.SliceToFormattedLinesWithPrefix(o.DNSNames, "DNS Name:"))
	return sb.String()
}
