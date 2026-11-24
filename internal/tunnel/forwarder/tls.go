package forwarder

import (
	"crypto/ecdsa"
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
