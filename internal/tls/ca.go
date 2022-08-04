package tls

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"time"
)

type CA struct {
	Key         *rsa.PrivateKey
	Certificate *x509.Certificate
	PEMs
}

type CAStore struct {
	Filename string
	Buffer   *bytes.Buffer
}

func NewCA() (*CA, error) {
	key, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("unable to generate rsa key: %w", err)
	}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"muting.io"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0),
		IsCA:      true,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(cryptorand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("unable to create certificate: %w", err)
	}

	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	}

	var b bytes.Buffer
	if err = pem.Encode(&b, block); err != nil {
		return nil, fmt.Errorf("unable to pem encode certificate: %w", err)
	}

	data, err := io.ReadAll(&b)
	if err != nil {
		return nil, fmt.Errorf("unable to read certificate: %w", err)
	}

	ca := &CA{
		Key:         key,
		Certificate: cert,
	}
	ca.PEMStores = map[string]PEMStore{
		"certificate": {Filename: "ca.crt", Data: data},
	}

	return ca, nil
}
