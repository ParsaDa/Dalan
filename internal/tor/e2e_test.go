package tor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
)

// TOR_E2E_BINARY is the tor executable inside an unpacked Tor Expert Bundle (<bundle>/tor/tor).
// It needs the internet, so the test skips when the variable is unset.
func TestTrialWithTheRealTorBinary(t *testing.T) {
	binary := os.Getenv("TOR_E2E_BINARY")
	if binary == "" {
		t.Skip("TOR_E2E_BINARY is not set")
	}
	country := os.Getenv("TOR_E2E_COUNTRY")
	if country == "" {
		country = "de"
	}
	layout := Layout{Root: t.TempDir(), BundleDir: filepath.Dir(filepath.Dir(binary))}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	got, err := Trial(ctx, country, nil, layout, tunnel.ProbeVia)
	if err != nil {
		t.Fatal(err)
	}
	if !got.OK || !strings.EqualFold(got.ExitLoc, country) {
		t.Fatalf("result = %+v, want a working exit in %s", got, country)
	}
}
