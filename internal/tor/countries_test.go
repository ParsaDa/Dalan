package tor

import (
	"strings"
	"testing"
)

func TestNormalizeCountry(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"DE", "de", true},
		{" nl ", "nl", true},
		{"gb", "gb", true},
		{"", "", false},
		{"d", "", false},
		{"deu", "", false},
		{"d1", "", false},
		{"zz-", "", false},
		{"{de}", "", false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, ok := NormalizeCountry(c.in)
			if got != c.want || ok != c.ok {
				t.Fatalf("NormalizeCountry(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
			}
		})
	}
}

func TestValidateBridges(t *testing.T) {
	obfs := "obfs4 192.0.2.4:443 0123456789ABCDEF0123456789ABCDEF01234567 cert=abc iat-mode=0"
	snow := "snowflake 192.0.2.3:80 2B280B23E1107BB62ABFC40DDCC8824814F80A72 fingerprint=2B280B23E1107BB62ABFC40DDCC8824814F80A72"
	plain := "192.0.2.5:9001 0123456789ABCDEF0123456789ABCDEF01234567"

	got, err := ValidateBridges("\n  " + obfs + "  \n\n" + snow + "\r\n" + plain + "\n")
	if err != nil {
		t.Fatalf("valid bridges refused: %v", err)
	}
	want := []string{obfs, snow, plain}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("ValidateBridges = %q, want %q", got, want)
	}

	bad := map[string]string{
		"carriage return inside a line": "obfs4 192.0.2.4:443 AAA\rExitPolicy accept *:*",
		"NUL byte":                      "obfs4 192.0.2.4:443 AAA\x00",
		"unknown transport":             "ExitPolicy accept *:*",
		"missing port":                  "192.0.2.5 0123",
		"directive as a bridge":         "Log debug file /etc/passwd",
	}
	for name, text := range bad {
		t.Run(name, func(t *testing.T) {
			_, err := ValidateBridges(text)
			if err == nil {
				t.Fatalf("ValidateBridges accepted %q", text)
			}
		})
	}
}

func TestValidateBridgesNamesTheBadLine(t *testing.T) {
	_, err := ValidateBridges("obfs4 192.0.2.4:443 AAA\nnonsense here")
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("error = %v, want one naming line 2", err)
	}
}
