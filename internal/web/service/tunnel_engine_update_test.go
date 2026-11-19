package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

// engineRelease builds a release whose engine writes the given connected state for its tunnel.
func engineRelease(t *testing.T, connected bool) engine.Release {
	t.Helper()
	return engineReleaseWithPadding(t, connected, 0)
}

// engineReleaseWithPadding adds incompressible bytes so the archive spans several transfer parts.
func engineReleaseWithPadding(t *testing.T, connected bool, padding int) engine.Release {
	t.Helper()
	script := `#!/bin/sh
name=$(basename "$2" .toml)
printf '{"name":"%s","connected":` + map[bool]string{true: "true", false: "false"}[connected] + `,"peer":"198.51.100.7:40000"}' "$name" > "$(dirname "$2")/$name.metrics.json"
exec sleep 600
`
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "backpack", Mode: 0o755, Size: int64(len(script)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	_, _ = tw.Write([]byte(script))
	if padding > 0 {
		filler := make([]byte, padding)
		_, _ = rand.Read(filler)
		_ = tw.WriteHeader(&tar.Header{Name: "padding.bin", Mode: 0o644, Size: int64(padding), Typeflag: tar.TypeReg})
		_, _ = tw.Write(filler)
	}
	tw.Close()
	gz.Close()
	sum := sha256.Sum256(buf.Bytes())
	sums := hex.EncodeToString(sum[:]) + "  backpack_linux_" + runtime.GOARCH + ".tar.gz\n"
	return engine.Release{Archive: buf.Bytes(), Sums: []byte(sums)}
}

func releaseFrom(rel engine.Release) engineSource {
	return func(context.Context, string, string, string) (engine.Release, error) { return rel, nil }
}

func TestAnEngineUpdateThatReconnectsTheTunnelsStaysInstalled(t *testing.T) {
	svc := newTunnelDB(t)
	sup := fakeBackpack(t)
	portal, err := svc.CreatePortal(backpackPortalSpec(), nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	waitForStatus(t, svc, portal.Id, true)

	if err := svc.updateEngineFrom(context.Background(), backpackEngine(), releaseFrom(engineRelease(t, true)), "v2.0.0"); err != nil {
		t.Fatalf("updateEngineFrom: %v", err)
	}
	if got := backpack.InstalledVersion(sup.Binary); got != "v2.0.0" {
		t.Fatalf("installed version = %q, want v2.0.0", got)
	}
	waitForStatus(t, svc, portal.Id, true)
}

func TestAnEngineUpdateWhoseTunnelsDoNotComeBackIsRolledBack(t *testing.T) {
	svc := newTunnelDB(t)
	sup := fakeBackpack(t)
	previousWait := engineSettleTimeout
	engineSettleTimeout = 600 * time.Millisecond
	t.Cleanup(func() { engineSettleTimeout = previousWait })
	portal, err := svc.CreatePortal(backpackPortalSpec(), nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	waitForStatus(t, svc, portal.Id, true)
	before, _ := os.ReadFile(sup.Binary)

	err = svc.updateEngineFrom(context.Background(), backpackEngine(), releaseFrom(engineRelease(t, false)), "v2.0.0")
	if err == nil || !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("updateEngineFrom error = %v, want one saying the update was rolled back", err)
	}
	if after, _ := os.ReadFile(sup.Binary); !bytes.Equal(before, after) {
		t.Fatal("the engine on disk is not the one from before the update")
	}
	waitForStatus(t, svc, portal.Id, true)
}

// serveReleaseFromNode makes the node side of an engine transfer answer with rel instead of going to GitHub.
func serveReleaseFromNode(t *testing.T, rel engine.Release) {
	t.Helper()
	previous := nodeEngineSource
	nodeEngineSource = func(context.Context, string, string, string) (engine.Release, error) { return rel, nil }
	engineDownloadsMu.Lock()
	engineDownloads = map[string]*engineDownload{}
	engineDownloadsMu.Unlock()
	t.Cleanup(func() { nodeEngineSource = previous })
}

func TestAnEngineCanBeInstalledThroughANodeInPartsWhenThisServerCannotReachGitHub(t *testing.T) {
	svc := newTunnelDB(t)
	sup := fakeBackpack(t)
	_, node := newFakeTunnelNode(t)
	rel := engineReleaseWithPadding(t, true, 3*backpack.ChunkSize+1234)
	rel.Version = "v2.0.0"
	if len(rel.Archive) < 3*backpack.ChunkSize {
		t.Fatalf("test archive is %d bytes, want it to span several parts", len(rel.Archive))
	}
	serveReleaseFromNode(t, rel)

	if err := svc.UpdateEngine(context.Background(), node.Id, "backpack", "", EngineRoute{}); err != nil {
		t.Fatalf("UpdateEngine through the node: %v", err)
	}
	if got := backpack.InstalledVersion(sup.Binary); got != "v2.0.0" {
		t.Fatalf("installed version = %q, want the node's v2.0.0", got)
	}
}

func TestANodeThatCannotDownloadTheEngineSaysWhy(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	_, node := newFakeTunnelNode(t)
	previous := nodeEngineSource
	nodeEngineSource = func(context.Context, string, string, string) (engine.Release, error) {
		return engine.Release{}, errors.New("the node cannot reach GitHub")
	}
	engineDownloadsMu.Lock()
	engineDownloads = map[string]*engineDownload{}
	engineDownloadsMu.Unlock()
	t.Cleanup(func() { nodeEngineSource = previous })

	if err := svc.UpdateEngine(context.Background(), node.Id, "backpack", "v3.0.0", EngineRoute{}); err == nil || !strings.Contains(err.Error(), "GitHub") {
		t.Fatalf("UpdateEngine when the node cannot download = %v, want the node's reason", err)
	}
}

func TestOnlyAPlainBackpackVersionIsAsked(t *testing.T) {
	svc := &TunnelService{}
	for _, version := range []string{"../../x", "v1.8.5/../../evil", "latest", "v1"} {
		if _, err := svc.EngineReleaseChunk("backpack", version, 0); err == nil || !strings.Contains(err.Error(), "version") {
			t.Errorf("EngineReleaseChunk(%q) error = %v, want a refusal of the version", version, err)
		}
		if err := svc.UpdateEngine(context.Background(), 0, "backpack", version, EngineRoute{}); err == nil || !strings.Contains(err.Error(), "version") {
			t.Errorf("UpdateEngine(%q) error = %v, want a refusal of the version", version, err)
		}
	}
}
