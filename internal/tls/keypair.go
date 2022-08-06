package tls

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"time"
)

type Keypair struct {
	Key         *rsa.PrivateKey
	Certificate *x509.Certificate
	PEMs
}

type KeypairOptions struct {
	CA         *CA
	CommonName string
	DNSNames   []string
}

type KeypairStore struct {
	Filename string
	Buffer   *bytes.Buffer
}

func NewKeypair(ctx context.Context, o KeypairOptions) (*Keypair, error) {
	key, err := rsa.GenerateKey(cryptorand.Reader, 1024)
	if err != nil {
		return nil, fmt.Errorf("unable to generate key: %w", err)
	}

	kblock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	var kb bytes.Buffer
	if err := pem.Encode(&kb, kblock); err != nil {
		return nil, fmt.Errorf("unable to pem encode private key: %w", err)
	}

	kdata, err := io.ReadAll(&kb)
	if err != nil {
		return nil, fmt.Errorf("unable to read key: %w", err)
	}

	cert := &x509.Certificate{
		DNSNames:     o.DNSNames,
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   o.CommonName,
			Organization: []string{"muting.io"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: subjectKeyID(key),
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage: x509.KeyUsageDigitalSignature,
	}

	der, err := x509.CreateCertificate(cryptorand.Reader, cert, o.CA.Certificate, &key.PublicKey, o.CA.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to create certificate: %w", err)
	}

	pblock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	}

	var cb bytes.Buffer
	if err := pem.Encode(&cb, pblock); err != nil {
		return nil, fmt.Errorf("unable to pem encode certificate: %w", err)
	}

	cdata, err := io.ReadAll(&cb)
	if err != nil {
		return nil, fmt.Errorf("unable to read key: %w", err)
	}

	keypair := &Keypair{
		Key:         key,
		Certificate: cert,
	}

	keypair.PEMStores = map[string]PEMStore{
		"key":         {Filename: "tls.key", Data: kdata},
		"certificate": {Filename: "tls.crt", Data: cdata},
	}

	return keypair, nil
}

func subjectKeyID(key *rsa.PrivateKey) []byte {
	b := x509.MarshalPKCS1PublicKey(&key.PublicKey)
	h := sha1.Sum(b)
	return h[:]
}
