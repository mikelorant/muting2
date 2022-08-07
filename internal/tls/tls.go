package tls

import (
	"context"
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

func NewTLS(ctx context.Context, o Options) (*TLS, error) {
	ca, err := NewCA(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to create new CA: %w", err)
	}

	keypair, err := NewKeypair(ctx, KeypairOptions{
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
	var strs []string
	strs = append(strs, fmt.Sprintf("Common Name: %v", o.CommonName))
	if str := format.SliceToFormattedLinesWithPrefix(o.DNSNames, "DNS Name:"); str != "" {
		strs = append(strs, str)
	}

	return strings.Join(strs, "\n")
}
