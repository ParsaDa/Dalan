package rakhsh

import (
	"encoding/base64"
	"strings"
	"testing"
)

func decode32(t *testing.T, label, value string) []byte {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("%s is not standard base64: %v", label, err)
	}
	if len(raw) != 32 {
		t.Fatalf("%s is %d bytes, want 32", label, len(raw))
	}
	return raw
}

func TestGenerateKeyPairDerivesItsPublicKey(t *testing.T) {
	secret, public, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	decode32(t, "secret", secret)
	decode32(t, "public", public)
	derived, err := PublicKey(secret)
	if err != nil {
		t.Fatal(err)
	}
	if derived != public {
		t.Fatalf("PublicKey(secret) = %q, want the pair's public key %q", derived, public)
	}
	other, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if other == secret {
		t.Fatal("two generated secrets are identical")
	}
}

// RFC 7748 section 6.1: Alice's private key and the public key it must derive.
func TestPublicKeyMatchesTheCurve25519Vector(t *testing.T) {
	secretHex := []byte{
		0x77, 0x07, 0x6d, 0x0a, 0x73, 0x18, 0xa5, 0x7d, 0x3c, 0x16, 0xc1, 0x72, 0x51, 0xb2, 0x66, 0x45,
		0xdf, 0x4c, 0x2f, 0x87, 0xeb, 0xc0, 0x99, 0x2a, 0xb1, 0x77, 0xfb, 0xa5, 0x1d, 0xb9, 0x2c, 0x2a,
	}
	want := []byte{
		0x85, 0x20, 0xf0, 0x09, 0x89, 0x30, 0xa7, 0x54, 0x74, 0x8b, 0x7d, 0xdc, 0xb4, 0x3e, 0xf7, 0x5a,
		0x0d, 0xbf, 0x3a, 0x0d, 0x26, 0x38, 0x1a, 0xf4, 0xeb, 0xa4, 0xa9, 0x8e, 0xaa, 0x9b, 0x4e, 0x6a,
	}
	got, err := PublicKey(base64.StdEncoding.EncodeToString(secretHex))
	if err != nil {
		t.Fatal(err)
	}
	if got != base64.StdEncoding.EncodeToString(want) {
		t.Fatalf("PublicKey = %q, want %q", got, base64.StdEncoding.EncodeToString(want))
	}
}

func TestPublicKeyRejectsAMalformedSecret(t *testing.T) {
	for name, secret := range map[string]string{
		"not base64": "not*base64",
		"too short":  base64.StdEncoding.EncodeToString(make([]byte, 31)),
		"empty":      "",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := PublicKey(secret)
			if err == nil || !strings.Contains(err.Error(), "secret") {
				t.Fatalf("PublicKey(%q) error = %v, want one that names the secret", secret, err)
			}
		})
	}
}
