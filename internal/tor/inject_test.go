package tor

import (
	"encoding/json"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/json_util"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

func outboundsOf(t *testing.T, cfg *xray.Config) []map[string]any {
	t.Helper()
	var out []map[string]any
	if err := json.Unmarshal(cfg.OutboundConfigs, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestApplyAddsEnabledExitsAsLoopbackSocksOutbounds(t *testing.T) {
	cfg := &xray.Config{OutboundConfigs: json_util.RawMessage(`[{"tag":"direct","protocol":"freedom"}]`)}
	exits := []model.TorExit{
		{Id: 1, Country: "de", Port: 62700, Enable: true},
		{Id: 2, Country: "nl", Port: 62701, Enable: false},
		{Id: 3, Country: "se", Port: 62702, Enable: true},
	}
	if err := Apply(cfg, exits); err != nil {
		t.Fatal(err)
	}
	out := outboundsOf(t, cfg)
	if len(out) != 3 || out[0]["tag"] != "direct" || out[1]["tag"] != "tor-de" || out[2]["tag"] != "tor-se" {
		t.Fatalf("outbounds = %v, want direct, tor-de, tor-se", out)
	}
	got, _ := json.Marshal(out[1])
	want := `{"protocol":"socks","settings":{"servers":[{"address":"127.0.0.1","port":62700}]},"tag":"tor-de"}`
	if string(got) != want {
		t.Fatalf("tor-de = %s, want %s", got, want)
	}
}

func TestApplyTurnsOnOutboundStatsSoTheTableShowsTraffic(t *testing.T) {
	cfg := &xray.Config{
		OutboundConfigs: json_util.RawMessage(`[]`),
		Policy:          json_util.RawMessage(`{"system":{"statsOutboundUplink":false,"statsOutboundDownlink":false}}`),
	}
	if err := Apply(cfg, []model.TorExit{{Country: "de", Port: 62700, Enable: true}}); err != nil {
		t.Fatal(err)
	}
	var raw map[string]map[string]any
	if err := json.Unmarshal(cfg.Policy, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["system"]["statsOutboundUplink"] != true || raw["system"]["statsOutboundDownlink"] != true {
		t.Fatalf("policy = %s, want outbound stats on", cfg.Policy)
	}
}

func TestApplyLeavesAConfigWithoutEnabledExitsByteIdentical(t *testing.T) {
	const outbounds = `[ {"tag":"direct",  "protocol":"freedom"} ]`
	cfg := &xray.Config{OutboundConfigs: json_util.RawMessage(outbounds)}
	if err := Apply(cfg, []model.TorExit{{Country: "nl", Port: 62701, Enable: false}}); err != nil {
		t.Fatal(err)
	}
	if string(cfg.OutboundConfigs) != outbounds {
		t.Fatalf("outbounds were rewritten to %s", cfg.OutboundConfigs)
	}
}

func TestApplyRefusesAnOutboundsSectionItCannotRead(t *testing.T) {
	cfg := &xray.Config{OutboundConfigs: json_util.RawMessage(`{not json`)}
	if err := Apply(cfg, []model.TorExit{{Country: "de", Port: 62700, Enable: true}}); err == nil {
		t.Fatal("Apply accepted an unreadable outbounds section")
	}
}
