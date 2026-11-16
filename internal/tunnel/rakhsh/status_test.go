package rakhsh

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// printsFixture is a stand-in binary that answers `status --socket S --json` with a file.
func printsFixture(t *testing.T, fixture string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rakhsh")
	script := "#!/bin/sh\n[ \"$1 $2 $4\" = \"status --socket --json\" ] || { echo \"unexpected arguments: $*\" >&2; exit 2; }\n[ \"$3\" = /run/dalan/r1.rakhsh.sock ] || { echo \"unexpected socket: $3\" >&2; exit 2; }\ncat " + fixture + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadStatusIsConnectedOnlyWhileALaneIsUp(t *testing.T) {
	idle, err := ReadStatus(context.Background(), printsFixture(t, "testdata/status-idle.json"), "/run/dalan/r1.rakhsh.sock")
	if err != nil {
		t.Fatal(err)
	}
	if idle.Connected || idle.LanesUp != 0 || idle.TargetLanes != 4 || idle.RTTMs != nil || idle.Peer != "" {
		t.Fatalf("a listening entry with no lanes = %+v, want not connected, 0 of 4 lanes and no link figures", idle)
	}

	up, err := ReadStatus(context.Background(), printsFixture(t, "testdata/status-up.json"), "/run/dalan/r1.rakhsh.sock")
	if err != nil {
		t.Fatal(err)
	}
	if !up.Connected || up.LanesUp != 3 || up.TargetLanes != 4 {
		t.Fatalf("status with three lanes up = %+v, want connected, 3 of 4", up)
	}
	if up.RTTMs == nil || *up.RTTMs != 48.5 || up.LossPct != 0.4 || up.Peer != "wss 203.0.113.10:7000" {
		t.Fatalf("link figures = %+v, want those of the first link (48.5 ms, 0.4%% loss, wss 203.0.113.10:7000)", up)
	}
}

func TestReadStatusWithoutTheBinaryIsNotInstalled(t *testing.T) {
	_, err := ReadStatus(context.Background(), filepath.Join(t.TempDir(), "rakhsh"), "/run/dalan/r1.rakhsh.sock")
	if !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("error = %v, want ErrNotInstalled", err)
	}
}

func TestReadStatusReportsWhatTheBinarySaidWhenItFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rakhsh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho \"cannot connect to the control socket\" >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := ReadStatus(context.Background(), path, "/run/dalan/r1.rakhsh.sock")
	if err == nil || !strings.Contains(err.Error(), "cannot connect to the control socket") {
		t.Fatalf("error = %v, want the binary's own message", err)
	}
}

func TestReadStatusRefusesAnswerThatIsNotStatusJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rakhsh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho 'not json'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadStatus(context.Background(), path, "/run/dalan/r1.rakhsh.sock"); err == nil {
		t.Fatal("an unreadable answer was taken for a status")
	}
}
