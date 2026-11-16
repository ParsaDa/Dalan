package rakhsh

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
)

// Keys the typed tunnel fields or other Dalan features own; options never name them.
var writtenFromTunnelFields = []string{
	"role", "token", "static_secret", "peer_public", "dial", "tunnel_listen", "tunnel_connect", "carrier", "control_socket",
}

// Router and panel-sync features that Xray and Dalan's own panel already cover.
var deliberatelyOmitted = []string{
	"access_log", "xui_panel", "xui_token", "xui_interval", "xui_target_host", "xui_proxy_protocol",
	"accept_sync", "sync_listen", "sync_ports",
}

func exampleTopLevelKeys(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile("testdata/rakhsh.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	assignment := regexp.MustCompile(`^#? ?([a-z][a-z0-9_]*) = `)
	// The OpenVPN banner opens the tables-only part of the file; its prose also looks like assignments.
	table := regexp.MustCompile(`^#? ?\[|^# ---- OpenVPN`)
	var keys []string
	for _, line := range strings.Split(string(raw), "\n") {
		if table.MatchString(line) {
			break
		}
		if m := assignment.FindStringSubmatch(line); m != nil && !slices.Contains(keys, m[1]) {
			keys = append(keys, m[1])
		}
	}
	return keys
}

func catalogKeys() []string {
	keys := make([]string, 0, len(Catalog))
	for _, o := range Catalog {
		keys = append(keys, o.Key)
	}
	return keys
}

func TestCatalogCoversEveryKeyOfTheExampleConfig(t *testing.T) {
	have := catalogKeys()
	for _, key := range exampleTopLevelKeys(t) {
		if slices.Contains(writtenFromTunnelFields, key) || slices.Contains(deliberatelyOmitted, key) {
			continue
		}
		if !slices.Contains(have, key) {
			t.Errorf("Rakhsh reads %q but the catalog does not offer it", key)
		}
	}
}

func TestEveryCatalogKeyIsOneRakhshReads(t *testing.T) {
	read := exampleTopLevelKeys(t)
	for _, o := range Catalog {
		if o.Kind == KindLinks {
			continue
		}
		if !slices.Contains(read, o.Key) {
			t.Errorf("catalog key %q is not in Rakhsh's reference config", o.Key)
		}
	}
}

func TestCatalogEntriesAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, o := range Catalog {
		if seen[o.Key] {
			t.Errorf("%q is listed twice", o.Key)
		}
		seen[o.Key] = true
		if o.HintEn == "" || o.HintFa == "" || o.Group == "" {
			t.Errorf("%q lacks a hint in both languages or a group", o.Key)
		}
		if o.Kind == KindChoice && len(o.Choices) == 0 {
			t.Errorf("%q is a choice with no choices", o.Key)
		}
		if o.Kind == KindInt && o.Max <= o.Min {
			t.Errorf("%q has an empty range %d..%d", o.Key, o.Min, o.Max)
		}
	}
}

func TestValidateOptionsRejects(t *testing.T) {
	cases := []struct {
		name    string
		options map[string]any
		want    string
	}{
		{"unknown key", map[string]any{"bogus": 1}, `option "bogus" is not offered for a reverse tunnel`},
		{"lanes below range", map[string]any{"lanes": 0}, `option "lanes": must be between 1 and 64`},
		{"preset outside the list", map[string]any{"preset": "fast"}, `option "preset": must be one of balance, turbo, aggressive, throughput`},
		{"rate without a unit", map[string]any{"limit_up": "100"}, `option "limit_up": is not a valid value for this option`},
		{"quota period", map[string]any{"quota_period": "year"}, `option "quota_period": must be one of day, week, month, never`},
		{"fec parity missing", map[string]any{"fec_data": 4}, `options "fec_data" and "fec_parity" must be set together`},
		{"l3 prefix too long", map[string]any{"l3_addr": "10.0.0.1/99"}, `option "l3_addr": must be an IP address with an optional prefix length`},
		{"text with a quote", map[string]any{"ws_credential": `a"b`}, `option "ws_credential": is not a valid value for this option`},
		{"bool as text", map[string]any{"resume": "yes"}, `option "resume": must be true or false`},
		{
			"link with an unknown carrier",
			map[string]any{"links": []any{map[string]any{"carrier": "bogus", "addr": "0.0.0.0:443"}}},
			`option "links": entry 1: unknown carrier "bogus"`,
		},
		{
			"two links on one address",
			map[string]any{"links": []any{
				map[string]any{"carrier": "quic", "addr": "0.0.0.0:443"},
				map[string]any{"carrier": "quic", "addr": "0.0.0.0:443"},
			}},
			`option "links": entries 1 and 2 use the same carrier and address`,
		},
		{
			"raw link without peer_ip",
			map[string]any{"links": []any{map[string]any{"carrier": "faketcp", "addr": "0.0.0.0:7001"}}},
			`option "links": entry 1: carrier "faketcp" needs the option peer_ip`,
		},
		{
			"link override that is not allowed",
			map[string]any{"links": []any{map[string]any{"carrier": "quic", "addr": "0.0.0.0:443", "lanes": 2}}},
			`option "links": entry 1: "lanes" cannot be set on a single link`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateOptions("reverse", tc.options)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateOptionsAcceptsARealisticSet(t *testing.T) {
	options := map[string]any{
		"encryption": "full", "resume": true, "resume_grace": 60.0, "lanes": 4, "preset": "turbo",
		"sni": "www.cloudflare.com", "ws_path": "/live", "ws_credential": "a-shared-string", "decoy": true,
		"tls_fingerprint": "chrome", "edge_ip": "1.1.1.1:443", "quic_mask": true, "congestion": "bbr",
		"link_policy": "spread", "link_timeout": 10, "link_probe": 30, "backup_connect": []any{"203.0.113.9:7443"},
		"fec_data": 10, "fec_parity": 2, "peer_ip": "203.0.113.9",
		"l3_addr": "10.77.0.1/30", "l3_mtu": 1380, "mss_clamp": 1300,
		"limit_up": "100mbit", "limit_down": "1.5Gbps", "limit_per_stream": "20 Mbit/s", "quota": "500GB", "quota_period": "month",
		"links": []any{
			map[string]any{"carrier": "quic", "addr": "0.0.0.0:7443", "sni": "www.microsoft.com", "quic_mask": true},
			map[string]any{"carrier": "faketcp", "addr": "0.0.0.0:7001", "faketcp_header": "linux"},
		},
	}
	for _, mode := range []string{"reverse", "direct"} {
		if err := ValidateOptions(mode, options); err != nil {
			t.Errorf("%s: %v", mode, err)
		}
	}
}

func TestOptionsForOffersEveryOptionInBothModes(t *testing.T) {
	if len(OptionsFor("reverse")) != len(Catalog) || len(OptionsFor("direct")) != len(Catalog) {
		t.Fatalf("reverse offers %d, direct %d, catalog has %d", len(OptionsFor("reverse")), len(OptionsFor("direct")), len(Catalog))
	}
	if len(OptionsFor("sideways")) != 0 {
		t.Fatal("an unknown mode should offer nothing")
	}
}

func TestRawCarriersAreCarriers(t *testing.T) {
	for _, c := range RawCarriers {
		if !slices.Contains(Carriers, c) {
			t.Errorf("raw carrier %q is not in Carriers", c)
		}
	}
}

func TestAnOptionReadsToTheFrontendLikeABackpackOptionBecauseTheApiDocumentsOnlyOne(t *testing.T) {
	keysOf := func(v any) []string {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}
	if got, want := keysOf(Option{}), keysOf(backpack.Option{}); !slices.Equal(got, want) {
		t.Fatalf("Rakhsh option fields %v differ from BackPack's %v; the OpenAPI schema \"Option\" describes both", got, want)
	}
}
