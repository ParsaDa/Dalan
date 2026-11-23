package service

import "testing"

func strPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

// The wizard saves name, country, address and "done" without sending any other setting.
func TestUpdateServerProfileWritesOnlyTheGivenFields(t *testing.T) {
	setupSettingTestDB(t)
	s := &SettingService{}

	err := s.UpdateServerProfile(ServerProfile{
		ServerName:    strPtr("  Tehran main "),
		ServerCountry: strPtr("IR"),
		ServerAddress: strPtr("panel.example.com"),
		SetupDone:     boolPtr(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateServerProfile(ServerProfile{Accent: strPtr("violet")}); err != nil {
		t.Fatal(err)
	}

	view, err := s.GetAllSettingView()
	if err != nil {
		t.Fatal(err)
	}
	if view.ServerName != "Tehran main" || view.ServerCountry != "IR" || view.ServerAddress != "panel.example.com" {
		t.Fatalf("profile = %q, %q, %q", view.ServerName, view.ServerCountry, view.ServerAddress)
	}
	if !view.SetupDone || view.Accent != "violet" {
		t.Fatalf("done = %v, accent = %q; want true and violet", view.SetupDone, view.Accent)
	}
}

// Skipping the wizard sends only "done" and must not wipe a name set earlier.
func TestUpdateServerProfileSkipKeepsTheName(t *testing.T) {
	setupSettingTestDB(t)
	s := &SettingService{}
	if err := s.UpdateServerProfile(ServerProfile{ServerName: strPtr("Frankfurt 1")}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateServerProfile(ServerProfile{SetupDone: boolPtr(true)}); err != nil {
		t.Fatal(err)
	}
	view, err := s.GetAllSettingView()
	if err != nil {
		t.Fatal(err)
	}
	if view.ServerName != "Frankfurt 1" || !view.SetupDone {
		t.Fatalf("name = %q, done = %v; want Frankfurt 1 and true", view.ServerName, view.SetupDone)
	}
}

// One bad value rejects the whole save, so the wizard never half-writes a profile.
func TestUpdateServerProfileRejectsBadValuesAndSavesNothing(t *testing.T) {
	setupSettingTestDB(t)
	s := &SettingService{}
	for name, p := range map[string]ServerProfile{
		"address with a scheme": {ServerName: strPtr("x"), ServerAddress: strPtr("https://example.com/panel")},
		"address with a port":   {ServerName: strPtr("x"), ServerAddress: strPtr("example.com:443")},
		"country name":          {ServerName: strPtr("x"), ServerCountry: strPtr("Iran")},
		"accent with a symbol":  {ServerName: strPtr("x"), Accent: strPtr("red;")},
	} {
		if err := s.UpdateServerProfile(p); err == nil {
			t.Fatalf("%s: saved, want an error", name)
		}
	}
	view, err := s.GetAllSettingView()
	if err != nil {
		t.Fatal(err)
	}
	if view.ServerName != "" {
		t.Fatalf("server name = %q after only rejected saves, want empty", view.ServerName)
	}
}

func TestValidServerAddress(t *testing.T) {
	for _, ok := range []string{"", "203.0.113.7", "2001:db8::1", "[2001:db8::1]", "panel.example.com", "localhost"} {
		if !validServerAddress(ok) {
			t.Errorf("%q rejected, want accepted", ok)
		}
	}
	for _, bad := range []string{"a b.com", "-bad.com", "exa_mple.com", "example..com", "http://example.com", "example.com/x"} {
		if validServerAddress(bad) {
			t.Errorf("%q accepted, want rejected", bad)
		}
	}
}
