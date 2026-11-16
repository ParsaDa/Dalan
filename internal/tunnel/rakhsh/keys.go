// Package rakhsh runs server tunnels on the Rakhsh engine: it writes the engine's
// configuration, installs its binary and reads the status it reports.
package rakhsh

import (
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

const keySize = 32

// GenerateKeyPair returns a new X25519 key pair as Rakhsh spells it: 32 bytes in standard base64.
func GenerateKeyPair() (secret, public string, err error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(key.Bytes()), base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()), nil
}

// PublicKey derives the public key of a secret key.
func PublicKey(secret string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("the secret key is not base64: %w", err)
	}
	if len(raw) != keySize {
		return "", errors.New("the secret key must be 32 bytes")
	}
	key, err := ecdh.X25519().NewPrivateKey(raw)
	if err != nil {
		return "", fmt.Errorf("the secret key is not valid: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()), nil
}

// BridgeSecret derives the exit end's secret key from the entry's key and the tunnel's uuid, so the portal
// can issue its setup link again without storing a second secret. It reveals nothing of the portal's key.
func BridgeSecret(portalSecret, tunnelUUID string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(portalSecret)
	if err != nil {
		return "", fmt.Errorf("the secret key is not base64: %w", err)
	}
	if len(raw) != keySize {
		return "", errors.New("the secret key must be 32 bytes")
	}
	mac := hmac.New(sha256.New, raw)
	mac.Write([]byte("dalan-rakhsh-bridge:" + tunnelUUID))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}
