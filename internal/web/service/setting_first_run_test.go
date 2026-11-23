package service

import "testing"

// The wizard stores the server's name and country and marks setup finished; all three survive a save.
func TestUpdateAllSettingKeepsFirstRunSettings(t *testing.T) {
	setupSettingTestDB(t)
	s := &SettingService{}

	view, err := s.GetAllSettingView()
	if err != nil {
		t.Fatal(err)
	}
	if view.SetupDone || view.ServerName != "" || view.ServerCountry != "" {
		t.Fatalf("fresh panel = done %v, name %q, country %q; want not done and empty", view.SetupDone, view.ServerName, view.ServerCountry)
	}

	settings := &view.AllSetting
	settings.ServerName = "Tehran main"
	settings.ServerCountry = "IR"
	settings.SetupDone = true
	if err := s.UpdateAllSetting(settings, SecretClears{}); err != nil {
		t.Fatal(err)
	}

	after, err := s.GetAllSettingView()
	if err != nil {
		t.Fatal(err)
	}
	if !after.SetupDone || after.ServerName != "Tehran main" || after.ServerCountry != "IR" {
		t.Fatalf("after save = done %v, name %q, country %q", after.SetupDone, after.ServerName, after.ServerCountry)
	}
}
