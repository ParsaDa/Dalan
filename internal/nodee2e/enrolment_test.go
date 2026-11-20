package nodee2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xtls/xray-core/common/geodata"
	"google.golang.org/protobuf/proto"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

// installXray gives a panel the Xray core that make node-e2e builds from go.mod's xray-core.
func (p *panel) installXray() {
	p.t.Helper()
	src := os.Getenv("XUI_NODE_E2E_XRAY")
	if src == "" {
		p.t.Fatal("XUI_NODE_E2E_XRAY is not set; run this through make node-e2e, which builds the Xray core")
	}
	in, err := os.Open(src)
	if err != nil {
		p.t.Fatal(err)
	}
	defer in.Close()
	out, err := os.OpenFile(filepath.Join(p.dir, "bin", xray.GetBinaryName()), os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		p.t.Fatal(err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		p.t.Fatal(err)
	}
	// The default routing blocks geoip:private, so Xray will not start without a geoip.dat that defines it.
	private, err := proto.Marshal(&geodata.GeoIPList{Entry: []*geodata.GeoIP{{Code: "PRIVATE", Cidr: []*geodata.CIDR{
		{Ip: []byte{127, 0, 0, 0}, Prefix: 8}, {Ip: []byte{10, 0, 0, 0}, Prefix: 8},
	}}}})
	if err != nil {
		p.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.dir, "bin", "geoip.dat"), private, 0o644); err != nil {
		p.t.Fatal(err)
	}
	p.moveXrayOffSharedPorts()
}

// Two cores on one host would fight over the template's fixed API and metrics ports.
func (p *panel) moveXrayOffSharedPorts() {
	p.t.Helper()
	template, err := os.ReadFile(filepath.Join("..", "web", "service", "config.json"))
	if err != nil {
		p.t.Fatal(err)
	}
	moved := strings.NewReplacer(
		"62789", strconv.Itoa(freePort(p.t)), "11111", strconv.Itoa(freePort(p.t)),
	).Replace(string(template))
	if moved == string(template) {
		p.t.Fatal("the Xray template no longer carries ports 62789 and 11111; update this helper")
	}
	sharedDBMu.Lock()
	defer sharedDBMu.Unlock()
	if err := database.InitDB(filepath.Join(p.dir, "db", "x-ui.db")); err != nil {
		p.t.Fatalf("%s: open db: %v", p.name, err)
	}
	db := database.GetDB()
	db.Exec("DELETE FROM settings WHERE key = ?", "xrayTemplateConfig")
	if err := db.Exec("INSERT INTO settings(key, value) VALUES (?, ?)", "xrayTemplateConfig", moved).Error; err != nil {
		p.t.Fatalf("%s: set xray template: %v", p.name, err)
	}
	if err := database.CloseDB(); err != nil {
		p.t.Fatalf("%s: close db: %v", p.name, err)
	}
}

type nodeRow struct {
	Id                  int    `json:"id"`
	Name                string `json:"name"`
	Address             string `json:"address"`
	Port                int    `json:"port"`
	AllowPrivateAddress bool   `json:"allowPrivateAddress"`
	Status              string `json:"status"`
	LastError           string `json:"lastError"`
}

func (p *panel) node(name string) (nodeRow, bool) {
	p.t.Helper()
	var rows []nodeRow
	if err := json.Unmarshal(p.call(http.MethodGet, "/panel/api/nodes/list", nil), &rows); err != nil {
		p.t.Fatalf("%s nodes: %v", p.name, err)
	}
	for _, r := range rows {
		if r.Name == name {
			return r, true
		}
	}
	return nodeRow{}, false
}

// installBackpack puts the BackPack engine next to a panel's Xray, where the panel looks for it.
func (p *panel) installBackpack(src string) {
	p.t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		p.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.dir, "bin", "backpack"), data, 0o755); err != nil {
		p.t.Fatal(err)
	}
}

// The master never dials the node: the only address it holds for it is its own loopback.
func TestANodeEnrolsByDiallingInAndIsManagedThroughItsTunnel(t *testing.T) {
	backpackBinary := os.Getenv("XUI_NODE_E2E_BACKPACK")
	if backpackBinary == "" {
		t.Skip("XUI_NODE_E2E_BACKPACK is not set; a node enrols over a BackPack tunnel")
	}
	bin := panelBinary(t)
	master := newPanel(t, bin, "enrol-master")
	node := newPanel(t, bin, "enrol-node")
	master.installXray()
	node.installXray()
	master.installBackpack(backpackBinary)
	node.installBackpack(backpackBinary)
	master.start()
	node.start()

	var invitation struct {
		Link string `json:"link"`
	}
	obj := master.call(http.MethodPost, "/panel/api/enrolments/add", map[string]any{
		"name": "abroad", "panelUrl": master.url() + "/", "address": "127.0.0.1", "port": freePort(t),
		"engine": "backpack", "transport": "tcpmux",
	})
	if err := json.Unmarshal(obj, &invitation); err != nil || !strings.HasPrefix(invitation.Link, "dalan://enrol/") {
		t.Fatalf("enrolments/add = %s (%v)", obj, err)
	}
	if _, ok := master.node("abroad"); ok {
		t.Fatalf("the node is listed before it enrolled")
	}

	node.cli("enrol", invitation.Link)
	// What "dalan enrol" does after the binary returns: restart Xray so it opens the tunnel.
	if err := node.cmd.Process.Signal(syscall.SIGUSR1); err != nil {
		t.Fatalf("signal node: %v", err)
	}

	row, ok := master.node("abroad")
	if !ok || row.Address != "127.0.0.1" || !row.AllowPrivateAddress || row.Port == node.port {
		t.Fatalf("master's node row = %+v (found %v), want a loopback address on a management port, not the node's own port %d", row, ok, node.port)
	}
	eventually(t, 120*time.Second, "master sees the node online through the tunnel", func() (bool, string) {
		r, _ := master.node("abroad")
		return r.Status == "online", fmt.Sprintf("status=%q lastError=%q", r.Status, r.LastError)
	})

	// Management really flows: an inbound created on the master for this node arrives on the node.
	port := freePort(t)
	master.call(http.MethodPost, "/panel/api/inbounds/add", vlessInbound("through-tunnel", port, &row.Id, vlessClient("tunnelled-user")))
	eventually(t, 90*time.Second, "the node receives an inbound pushed through the tunnel", func() (bool, string) {
		ib, found := node.inboundOnPort(port)
		return found && hasAll(ib.emails(), "tunnelled-user"), fmt.Sprintf("found=%v", found)
	})

	again := exec.Command(node.bin, "enrol", invitation.Link)
	again.Env = node.env()
	out, err := again.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "not valid") {
		t.Fatalf("a second enrol with the same link: err=%v output=%s; want a refusal", err, out)
	}
}
