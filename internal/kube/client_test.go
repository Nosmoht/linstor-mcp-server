package kube

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func TestDecodeB64(t *testing.T) {
	got, err := DecodeB64("YQ==")
	if err != nil {
		t.Fatalf("DecodeB64() error = %v", err)
	}
	if string(got) != "a" {
		t.Fatalf("DecodeB64() = %q, want a", string(got))
	}
}

func TestTLSCertificateInvalid(t *testing.T) {
	if _, err := TLSCertificate([]byte("bad"), []byte("bad")); err == nil {
		t.Fatal("TLSCertificate() error = nil, want error")
	}
}

func TestTLSCertificateValid(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-cert"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey() error = %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	if _, err := TLSCertificate(certPEM, keyPEM); err != nil {
		t.Fatalf("TLSCertificate() error = %v", err)
	}
}
