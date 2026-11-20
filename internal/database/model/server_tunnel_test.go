package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExternalEngineNamesTheEnginesRunAsOwnProcesses(t *testing.T) {
	for engine, want := range map[string]bool{
		"xray":               false,
		"":                   false,
		TunnelEngineBackpack: true,
		TunnelEngineRakhsh:   true,
	} {
		if got := ExternalEngine(engine); got != want {
			t.Errorf("ExternalEngine(%q) = %v, want %v", engine, got, want)
		}
	}
}

func TestRakhshSecretNeverLeavesInJSON(t *testing.T) {
	raw, err := json.Marshal(ServerTunnel{
		Name: "r1", Engine: TunnelEngineRakhsh, RakhshSecret: "very-secret", RakhshPeerPublic: "the-public-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "very-secret") {
		t.Fatalf("the Rakhsh secret key is in the tunnel JSON: %s", text)
	}
	if !strings.Contains(text, "the-public-key") {
		t.Fatalf("the peer's public key should be listed: %s", text)
	}
}
