package rakhsh

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Status is the part of `rakhsh status` the panel shows. The link figures are those of the first link,
// the one lanes prefer.
type Status struct {
	Connected   bool     `json:"connected"`
	LanesUp     int      `json:"lanesUp"`
	TargetLanes int      `json:"targetLanes"`
	RTTMs       *float64 `json:"rttMs"`
	LossPct     float64  `json:"lossPct"`
	Peer        string   `json:"peer"`
}

// ReadStatus asks a running Rakhsh process for its status over its control socket.
func ReadStatus(ctx context.Context, binary, socket string) (Status, error) {
	if _, err := os.Stat(binary); err != nil {
		return Status{}, ErrNotInstalled
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, binary, "status", "--socket", socket, "--json")
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if said := strings.TrimSpace(stderr.String()); said != "" {
			return Status{}, fmt.Errorf("rakhsh status: %s", said)
		}
		return Status{}, fmt.Errorf("rakhsh status: %w", err)
	}
	var snap struct {
		LanesUp     int `json:"lanes_up"`
		TargetLanes int `json:"target_lanes"`
		Links       []struct {
			Label   string   `json:"label"`
			RTTMs   *float64 `json:"rtt_ms"`
			LossPct float64  `json:"loss_pct"`
		} `json:"links"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &snap); err != nil {
		return Status{}, errors.New("rakhsh status: the answer is not a status report")
	}
	st := Status{Connected: snap.LanesUp > 0, LanesUp: snap.LanesUp, TargetLanes: snap.TargetLanes}
	if len(snap.Links) > 0 {
		first := snap.Links[0]
		st.Peer, st.RTTMs, st.LossPct = first.Label, first.RTTMs, first.LossPct
	}
	return st, nil
}
