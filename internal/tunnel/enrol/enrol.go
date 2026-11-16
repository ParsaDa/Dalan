// Package enrol is the exchange by which a node abroad joins a main panel it must dial into.
package enrol

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	linkPrefix  = "dalan://enrol/"
	linkVersion = 1
	// RedeemPath is where a panel accepts enrolments, relative to its base path.
	RedeemPath = "dalan/enrol"

	maxReplyBytes = 64 << 10
	aadRequest    = "dalan enrol request"
	aadReply      = "dalan enrol reply"
)

// ErrSealed means a body was not sealed for this enrolment; it deliberately says nothing more.
var ErrSealed = errors.New("enrolment message could not be opened")

var errNotALink = errors.New("not a Dalan enrolment link")

// Offer is what a panel keeps for one enrolment; only Link leaves the panel.
type Offer struct {
	ID         string
	Token      string
	PrivateKey string
	Link       string
}

// Link is an enrolment link as the node reads it.
type Link struct {
	URL   string
	ID    string
	Token string
	Key   string
}

type linkPayload struct {
	Version int    `json:"v"`
	URL     string `json:"url"`
	ID      string `json:"id"`
	Token   string `json:"token"`
	Key     string `json:"key"`
}

// NodeInfo is what a node tells the panel about itself. Token proves it holds the link.
type NodeInfo struct {
	Token    string `json:"token"`
	Port     int    `json:"port"`
	BasePath string `json:"basePath"`
	Scheme   string `json:"scheme"`
	APIToken string `json:"apiToken"`
}

// Grant is the panel's answer: the dialling end of the node's management tunnel, as a tunnel setup link,
// and the dialling end of the node's own traffic tunnel when the invitation asked for one.
type Grant struct {
	Tunnel     string `json:"tunnel"`
	DataTunnel string `json:"dataTunnel,omitempty"`
}

type wireRequest struct {
	ID  string `json:"id"`
	Key string `json:"key"`
	Box string `json:"box"`
}

type wireReply struct {
	Box string `json:"box"`
}

var b64 = base64.RawURLEncoding

func randomString(n int) (string, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return b64.EncodeToString(raw), nil
}

// NewOffer makes a one-time enrolment for the panel reachable at panelURL.
func NewOffer(panelURL string) (Offer, error) {
	u, err := url.Parse(strings.TrimSpace(panelURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Offer{}, fmt.Errorf("panel address %q must be an http or https URL", panelURL)
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	u.RawQuery, u.Fragment = "", ""
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return Offer{}, err
	}
	id, err := randomString(12)
	if err != nil {
		return Offer{}, err
	}
	token, err := randomString(32)
	if err != nil {
		return Offer{}, err
	}
	raw, err := json.Marshal(linkPayload{
		Version: linkVersion, URL: u.String(), ID: id, Token: token, Key: b64.EncodeToString(key.PublicKey().Bytes()),
	})
	if err != nil {
		return Offer{}, err
	}
	return Offer{
		ID: id, Token: token, PrivateKey: b64.EncodeToString(key.Bytes()),
		Link: linkPrefix + b64.EncodeToString(raw),
	}, nil
}

// ParseLink reads an enrolment link.
func ParseLink(link string) (Link, error) {
	encoded, ok := strings.CutPrefix(strings.TrimSpace(link), linkPrefix)
	if !ok {
		return Link{}, errNotALink
	}
	raw, err := b64.DecodeString(encoded)
	if err != nil {
		return Link{}, errNotALink
	}
	var p linkPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return Link{}, errNotALink
	}
	if p.Version != linkVersion {
		return Link{}, fmt.Errorf("enrolment link is version %d; this panel understands version %d", p.Version, linkVersion)
	}
	if p.URL == "" || p.ID == "" || p.Token == "" || p.Key == "" {
		return Link{}, errNotALink
	}
	return Link{URL: p.URL, ID: p.ID, Token: p.Token, Key: p.Key}, nil
}

func sessionCipher(private *ecdh.PrivateKey, peer string) (cipher.AEAD, error) {
	peerRaw, err := b64.DecodeString(peer)
	if err != nil {
		return nil, ErrSealed
	}
	peerKey, err := ecdh.X25519().NewPublicKey(peerRaw)
	if err != nil {
		return nil, ErrSealed
	}
	shared, err := private.ECDH(peerKey)
	if err != nil {
		return nil, ErrSealed
	}
	key, err := hkdf.Key(sha256.New, shared, nil, "dalan enrol v1", 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func seal(aead cipher.AEAD, aad string, v any) (string, error) {
	plain, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return b64.EncodeToString(aead.Seal(nonce, nonce, plain, []byte(aad))), nil
}

func open(aead cipher.AEAD, aad, box string, v any) error {
	raw, err := b64.DecodeString(box)
	if err != nil || len(raw) < aead.NonceSize() {
		return ErrSealed
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte(aad))
	if err != nil {
		return ErrSealed
	}
	if err := json.Unmarshal(plain, v); err != nil {
		return ErrSealed
	}
	return nil
}

// Seal makes the request body a node sends and returns the function that opens the panel's reply.
func (l Link) Seal(info NodeInfo) ([]byte, func(reply []byte) (Grant, error), error) {
	private, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	aead, err := sessionCipher(private, l.Key)
	if err != nil {
		return nil, nil, errNotALink
	}
	box, err := seal(aead, aadRequest, info)
	if err != nil {
		return nil, nil, err
	}
	body, err := json.Marshal(wireRequest{ID: l.ID, Key: b64.EncodeToString(private.PublicKey().Bytes()), Box: box})
	if err != nil {
		return nil, nil, err
	}
	openReply := func(reply []byte) (Grant, error) {
		var wire wireReply
		if err := json.Unmarshal(reply, &wire); err != nil {
			return Grant{}, ErrSealed
		}
		var grant Grant
		if err := open(aead, aadReply, wire.Box, &grant); err != nil {
			return Grant{}, err
		}
		return grant, nil
	}
	return body, openReply, nil
}

// RequestID is the public identifier of the enrolment a request body claims, read before anything is opened.
func RequestID(body []byte) (string, error) {
	var wire wireRequest
	if err := json.Unmarshal(body, &wire); err != nil || wire.ID == "" {
		return "", ErrSealed
	}
	return wire.ID, nil
}

// OpenRequest opens a node's request with the offer's private key and returns the function that seals the reply.
func OpenRequest(privateKey string, body []byte) (NodeInfo, func(Grant) ([]byte, error), error) {
	var wire wireRequest
	if err := json.Unmarshal(body, &wire); err != nil {
		return NodeInfo{}, nil, ErrSealed
	}
	rawKey, err := b64.DecodeString(privateKey)
	if err != nil {
		return NodeInfo{}, nil, ErrSealed
	}
	private, err := ecdh.X25519().NewPrivateKey(rawKey)
	if err != nil {
		return NodeInfo{}, nil, ErrSealed
	}
	aead, err := sessionCipher(private, wire.Key)
	if err != nil {
		return NodeInfo{}, nil, err
	}
	var info NodeInfo
	if err := open(aead, aadRequest, wire.Box, &info); err != nil {
		return NodeInfo{}, nil, err
	}
	sealReply := func(grant Grant) ([]byte, error) {
		box, err := seal(aead, aadReply, grant)
		if err != nil {
			return nil, err
		}
		return json.Marshal(wireReply{Box: box})
	}
	return info, sealReply, nil
}

// Redeem sends a node's details to the panel named in the link and returns the tunnel it was granted.
func Redeem(ctx context.Context, client *http.Client, link Link, info NodeInfo) (Grant, error) {
	body, openReply, err := link.Seal(info)
	if err != nil {
		return Grant{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, link.URL+RedeemPath, bytes.NewReader(body))
	if err != nil {
		return Grant{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Grant{}, fmt.Errorf("the main panel could not be reached: %w", err)
	}
	defer resp.Body.Close()
	reply, err := io.ReadAll(io.LimitReader(resp.Body, maxReplyBytes))
	if err != nil {
		return Grant{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Grant{}, fmt.Errorf("the main panel refused the enrolment: %s", strings.TrimSpace(string(reply)))
	}
	return openReply(reply)
}
