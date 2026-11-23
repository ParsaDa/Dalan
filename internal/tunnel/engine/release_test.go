package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func releaseListServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/engine/releases" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestRecentTagsListsTheNewestPlainReleases(t *testing.T) {
	server := releaseListServer(t, `[
		{"tag_name": "v2.0.0-rc1", "prerelease": true},
		{"tag_name": "v1.9.0"},
		{"tag_name": "nightly"},
		{"tag_name": "v1.8.5", "draft": true},
		{"tag_name": "v1.8.4"},
		{"tag_name": "v1.8.3"},
		{"tag_name": "v1.8.2"},
		{"tag_name": "v1.8.1"}
	]`, http.StatusOK)

	got, err := RecentTags(context.Background(), server.Client(), server.URL, "owner/engine", "Test", 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"v1.9.0", "v1.8.4", "v1.8.3", "v1.8.2"}
	if !slices.Equal(got, want) {
		t.Fatalf("RecentTags = %v, want %v", got, want)
	}
}

func TestRecentTagsReportsAnUnreachableOrEmptyList(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		status int
		want   string
	}{
		{"refused", `{"message": "rate limited"}`, http.StatusForbidden, "status 403"},
		{"nothing installable", `[{"tag_name": "nightly"}]`, http.StatusOK, "no Test release"},
		{"not a list", `{"message": "oops"}`, http.StatusOK, "could not be read"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server := releaseListServer(t, c.body, c.status)
			_, err := RecentTags(context.Background(), server.Client(), server.URL, "owner/engine", "Test", 4)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("RecentTags error = %v, want one containing %q", err, c.want)
			}
		})
	}
}
