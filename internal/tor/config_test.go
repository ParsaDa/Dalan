package tor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func testLayout() Layout {
	return Layout{Root: "/var/x-ui/tor", BundleDir: "/opt/x-ui/bin/tor-bundle"}
}

func TestRenderConfigWithoutBridges(t *testing.T) {
	got, err := RenderConfig(model.TorExit{Country: "de", Port: 62700, Enable: true}, nil, testLayout())
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		`DataDirectory "/var/x-ui/tor/data/de"`,
		`GeoIPFile "/opt/x-ui/bin/tor-bundle/data/geoip"`,
		`GeoIPv6File "/opt/x-ui/bin/tor-bundle/data/geoip6"`,
		`SocksPort 127.0.0.1:62700`,
		`ClientOnly 1`,
		`ExitNodes {de}`,
		`StrictNodes 1`,
		`Log notice stdout`,
		``,
	}, "\n")
	if got != want {
		t.Fatalf("RenderConfig =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderConfigWithBridgesNamesOnlyTheTransportsInUse(t *testing.T) {
	bridges := []string{
		"obfs4 192.0.2.4:443 0123456789ABCDEF0123456789ABCDEF01234567 cert=abc iat-mode=0",
		"snowflake 192.0.2.3:80 2B280B23E1107BB62ABFC40DDCC8824814F80A72",
	}
	got, err := RenderConfig(model.TorExit{Country: "nl", Port: 62701, Enable: true}, bridges, testLayout())
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"UseBridges 1",
		"Bridge " + bridges[0],
		"Bridge " + bridges[1],
		"ClientTransportPlugin obfs4,meek_lite,webtunnel exec /opt/x-ui/bin/tor-bundle/tor/pluggable_transports/lyrebird",
		"ClientTransportPlugin snowflake exec /opt/x-ui/bin/tor-bundle/tor/pluggable_transports/snowflake-client",
	} {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("config lacks the line %q:\n%s", line, got)
		}
	}

	only, err := RenderConfig(model.TorExit{Country: "nl", Port: 62701}, bridges[:1], testLayout())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(only, "snowflake") {
		t.Fatalf("an obfs4-only config mentions snowflake:\n%s", only)
	}
}

func TestRenderConfigNeverBindsBeyondLoopback(t *testing.T) {
	for port := PortBase; port <= PortLimit; port += 7 {
		got, err := RenderConfig(model.TorExit{Country: "se", Port: port}, nil, testLayout())
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(got, "\n") {
			if strings.HasPrefix(line, "SocksPort ") && line != fmt.Sprintf("SocksPort 127.0.0.1:%d", port) {
				t.Fatalf("port %d: %q is not a loopback-only listener", port, line)
			}
		}
	}
}

func TestRenderConfigRefusesWhatItCouldNotWriteSafely(t *testing.T) {
	cases := map[string]struct {
		exit    model.TorExit
		bridges []string
		layout  Layout
	}{
		"country with a brace": {model.TorExit{Country: "d}", Port: 62700}, nil, testLayout()},
		"three-letter country": {model.TorExit{Country: "deu", Port: 62700}, nil, testLayout()},
		"port zero":            {model.TorExit{Country: "de", Port: 0}, nil, testLayout()},
		"port above the range": {model.TorExit{Country: "de", Port: 70000}, nil, testLayout()},
		"bridge path with a space": {
			model.TorExit{Country: "de", Port: 62700},
			[]string{"obfs4 192.0.2.4:443 AAA"},
			Layout{Root: "/var/x-ui/tor", BundleDir: "/opt/my bundle"},
		},
		"newline in the data path": {model.TorExit{Country: "de", Port: 62700}, nil, Layout{Root: "/var/x\nExitPolicy accept *:*", BundleDir: "/b"}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got, err := RenderConfig(c.exit, c.bridges, c.layout); err == nil {
				t.Fatalf("RenderConfig accepted it:\n%s", got)
			}
		})
	}
}
