package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

// redirectingClient sends every request, whatever its host, to server.
func redirectingClient(server *httptest.Server) *http.Client {
	target, _ := url.Parse(server.URL)
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme, r.URL.Host = target.Scheme, target.Host
		return http.DefaultTransport.RoundTrip(r)
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func useListClient(t *testing.T, client *http.Client) {
	t.Helper()
	previous := engineListClient
	engineListClient = func(EngineRoute) (*http.Client, func(), error) { return client, func() {}, nil }
	t.Cleanup(func() { engineListClient = previous })
}

func TestEngineReleasesListsTheNewestFourOfTheEngineThatWasAsked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"tag_name":"v1.9.0"},{"tag_name":"v1.8.5"},{"tag_name":"v1.8.4"},{"tag_name":"v1.8.3"},{"tag_name":"v1.8.2"}]`)
	}))
	t.Cleanup(server.Close)
	useListClient(t, redirectingClient(server))

	for _, id := range []string{"backpack", "rakhsh"} {
		got, err := (&TunnelService{}).EngineReleases(context.Background(), 0, id, EngineRoute{})
		if err != nil {
			t.Fatalf("EngineReleases(%s): %v", id, err)
		}
		if want := []string{"v1.9.0", "v1.8.5", "v1.8.4", "v1.8.3"}; !slices.Equal(got, want) {
			t.Fatalf("EngineReleases(%s) = %v, want %v", id, got, want)
		}
	}
}

func TestEngineReleasesNamesTheEngineWhenGitHubCannotBeReached(t *testing.T) {
	proxy, seen := recordingProxy(t)
	_, err := (&TunnelService{}).EngineReleases(context.Background(), 0, "rakhsh", EngineRoute{Via: RouteLink, Value: proxy.URL})
	if err == nil || !strings.Contains(err.Error(), "could not list the Rakhsh releases") {
		t.Fatalf("err = %v, want one saying the Rakhsh releases could not be listed", err)
	}
	if hosts := seen(); len(hosts) == 0 || !strings.Contains(hosts[0], "api.github.com") {
		t.Fatalf("the proxy saw %v, want the GitHub API", hosts)
	}
}

func TestEngineReleasesRefusesAnUnknownEngine(t *testing.T) {
	if _, err := (&TunnelService{}).EngineReleases(context.Background(), 0, "xray", EngineRoute{}); err == nil {
		t.Fatal("an engine the panel does not run was listed")
	}
}

func TestInstallEngineVersionDownloadsTheReleaseThatWasPicked(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	var asked string
	previous := engineSourceFor
	engineSourceFor = func(EngineRoute) (engineSource, func(), error) {
		return func(_ context.Context, _, version, _ string) (engine.Release, error) {
			asked = version
			return engineRelease(t, true), nil
		}, func() {}, nil
	}
	t.Cleanup(func() { engineSourceFor = previous })

	if err := svc.InstallEngineVersion("backpack", "v1.8.2", EngineRoute{}); err != nil {
		t.Fatalf("InstallEngineVersion: %v", err)
	}
	if asked != "v1.8.2" {
		t.Fatalf("the source was asked for %q, want the picked v1.8.2", asked)
	}
	if err := svc.InstallEngineVersion("backpack", "", EngineRoute{}); err != nil {
		t.Fatalf("InstallEngineVersion without a pick: %v", err)
	}
	if asked != backpackEngine().pinned {
		t.Fatalf("with no pick the source was asked for %q, want the pinned %q", asked, backpackEngine().pinned)
	}
}

func TestInstallEngineVersionRefusesATagThatIsNotAVersionBeforeDownloading(t *testing.T) {
	previous := engineSourceFor
	engineSourceFor = func(EngineRoute) (engineSource, func(), error) {
		t.Fatal("a download was opened for a tag that is not a version")
		return nil, nil, nil
	}
	t.Cleanup(func() { engineSourceFor = previous })

	err := (&TunnelService{}).InstallEngineVersion("rakhsh", "../../evil", EngineRoute{})
	if err == nil || !strings.Contains(err.Error(), "not a Rakhsh version") {
		t.Fatalf("err = %v, want a refusal naming the Rakhsh version", err)
	}
}

func TestEngineReleasesAreReadByTheNodeWhenOneIsChosen(t *testing.T) {
	svc := newTunnelDB(t)
	fake, node := newFakeTunnelNode(t)
	useListClient(t, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("this server read the list itself although a node was chosen")
		return nil, fmt.Errorf("unreachable")
	})})

	got, err := svc.EngineReleases(context.Background(), node.Id, "backpack", EngineRoute{})
	if err != nil {
		t.Fatalf("EngineReleases through the node: %v", err)
	}
	if want := []string{"v1.9.0", "v1.8.5"}; !slices.Equal(got, want) {
		t.Fatalf("EngineReleases = %v, want the node's list %v", got, want)
	}
	if !slices.Contains(fake.calls, "POST /panel/api/tunnels/engine/releases") {
		t.Fatalf("the node was called %v, want its releases endpoint", fake.calls)
	}
}
