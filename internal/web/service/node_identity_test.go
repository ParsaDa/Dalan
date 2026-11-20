package service

import (
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestNodeCountryAndAccentSurviveCreateAndUpdate(t *testing.T) {
	setupConflictDB(t)

	svc := &NodeService{}
	token := "tok"
	view, err := svc.CreateFromRequest(&NodeMutationRequest{
		Name: "nl-ams-1", Scheme: "http", Address: "127.0.0.1", Port: 61000,
		ApiToken: &token, Enable: true, AllowPrivateAddress: true,
		Country: " nl ", Accent: "Tulip",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Country != "NL" || view.Accent != "tulip" {
		t.Fatalf("created country/accent = %q/%q, want NL/tulip", view.Country, view.Accent)
	}

	err = svc.UpdateFromRequest(view.Id, &NodeMutationRequest{
		Name: "nl-ams-1", Scheme: "http", Address: "127.0.0.1", Port: 61000,
		Enable: true, AllowPrivateAddress: true, Country: "DE", Accent: "#34D399",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := svc.GetViewById(view.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Country != "DE" || got.Accent != "#34d399" {
		t.Fatalf("updated country/accent = %q/%q, want DE/#34d399", got.Country, got.Accent)
	}
}

func TestNodeCountryAndAccentAreValidated(t *testing.T) {
	tests := []struct {
		name    string
		country string
		accent  string
		wantErr bool
	}{
		{"both empty", "", "", false},
		{"preset accent", "JP", "rose", false},
		{"custom hex accent", "", "#1a2b3c", false},
		{"three-letter country", "NLD", "", true},
		{"digit in country", "N1", "", true},
		{"unknown accent name", "", "teal", true},
		{"short hex", "", "#fff", true},
		{"hex without hash", "", "34d399", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := &model.Node{Name: "n", Address: "example.com", Port: 2053, Country: tc.country, Accent: tc.accent}
			err := (&NodeService{}).normalize(n)
			if (err != nil) != tc.wantErr {
				t.Fatalf("normalize(country=%q, accent=%q) error = %v, wantErr %v", tc.country, tc.accent, err, tc.wantErr)
			}
		})
	}
}
