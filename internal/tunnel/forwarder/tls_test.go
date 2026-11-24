package forwarder

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"testing"
)

func TestDeriveCertGivesBothEndsTheSameCertificateFromOneSecret(t *testing.T) {
	a, err := DeriveCert("one-secret-of-thirty-two-characters")
	if err != nil {
		t.Fatal(err)
	}
	b, err := DeriveCert("one-secret-of-thirty-two-characters")
	if err != nil {
		t.Fatal(err)
	}
	other, err := DeriveCert("another-secret-of-thirty-two-char")
	if err != nil {
		t.Fatal(err)
	}
	if string(a.CertPEM) != string(b.CertPEM) || a.Pin != b.Pin {
		t.Fatal("the same secret gave two different certificates")
	}
	if a.Pin == other.Pin {
		t.Fatal("two secrets gave the same certificate")
	}
	block, _ := pem.Decode(a.CertPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(cert.Raw)
	if hex.EncodeToString(sum[:]) != a.Pin {
		t.Fatalf("pin %s is not the SHA-256 of the certificate", a.Pin)
	}
}
