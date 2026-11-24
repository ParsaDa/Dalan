package forwarder

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// SelfSigned makes a certificate and key for the listening end of a TLS transport, or reads the ones it
// made before; the dialing end does not verify it, the shared secret is what authenticates the peer.
func SelfSigned(dir, name string) (certFile, keyFile string, err error) {
	certFile = filepath.Join(dir, name+".crt")
	keyFile = filepath.Join(dir, name+".key")
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			return certFile, keyFile, nil
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return "", "", err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(10, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		_ = os.Remove(certFile)
		return "", "", err
	}
	return certFile, keyFile, nil
}

var errNoDir = errors.New("this transport needs a folder for its certificate")

// certFor is the certificate files of a tunnel end that listens over TLS.
func certFor(p Plan) (certFile, keyFile string, err error) {
	if p.Dir == "" {
		return "", "", errNoDir
	}
	return SelfSigned(p.Dir, p.Name)
}

// Fingerprint is the SHA-256 of a certificate file's first block, as an engine that pins certificates wants it.
func Fingerprint(certFile string) (string, error) {
	raw, err := os.ReadFile(certFile)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return "", fmt.Errorf("%s holds no certificate", certFile)
	}
	sum := sha256.Sum256(block.Bytes)
	return hex.EncodeToString(sum[:]), nil
}

// Derived is a certificate and key both ends of a tunnel compute from its shared secret, so the dialing
// end can pin the certificate without ever receiving a file. Pin is the SHA-256 of its DER form in hex.
type Derived struct {
	CertPEM, KeyPEM []byte
	Pin             string
}

// DeriveCert makes the same certificate for the same secret every time: an Ed25519 key from a hash of the
// secret, a fixed serial and fixed validity.
func DeriveCert(secret string) (Derived, error) {
	seed := sha256.Sum256([]byte("dalan-tls-key:" + secret))
	key := ed25519.NewKeyFromSeed(seed[:])
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"},
		NotBefore: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), NotAfter: time.Date(2120, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(nil, tmpl, tmpl, key.Public(), key)
	if err != nil {
		return Derived{}, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return Derived{}, err
	}
	sum := sha256.Sum256(der)
	return Derived{
		CertPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		KeyPEM:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}),
		Pin:     hex.EncodeToString(sum[:]),
	}, nil
}

// derivedFiles writes the derived certificate of a plan's secret into its folder and returns the paths.
func derivedFiles(p Plan) (certFile, keyFile string, err error) {
	if p.Dir == "" {
		return "", "", errNoDir
	}
	d, err := DeriveCert(p.Secret)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return "", "", err
	}
	certFile, keyFile = filepath.Join(p.Dir, p.Name+".crt"), filepath.Join(p.Dir, p.Name+".key")
	if err := os.WriteFile(certFile, d.CertPEM, 0o600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(keyFile, d.KeyPEM, 0o600); err != nil {
		return "", "", err
	}
	return certFile, keyFile, nil
}
