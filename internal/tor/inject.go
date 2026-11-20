package tor

import (
	"encoding/json"
	"fmt"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/tunnelroute"
	"github.com/mhsanaei/3x-ui/v3/internal/util/json_util"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

// Apply adds every enabled exit to a generated config as a loopback SOCKS outbound; the stored template is never touched.
func Apply(cfg *xray.Config, exits []model.TorExit) error {
	var added []any
	for _, exit := range exits {
		if exit.Enable {
			added = append(added, exitOutbound(exit))
		}
	}
	// Re-marshalling reorders keys, so an untouched config must stay byte-identical for hot-apply diffs.
	if len(added) == 0 {
		return nil
	}
	var outbounds []any
	if len(cfg.OutboundConfigs) > 0 {
		if err := json.Unmarshal(cfg.OutboundConfigs, &outbounds); err != nil {
			return fmt.Errorf("tor: outbounds section is unparsable: %w", err)
		}
	}
	if err := tunnelroute.EnableOutboundStats(cfg); err != nil {
		return err
	}
	raw, err := json.Marshal(append(outbounds, added...))
	if err != nil {
		return err
	}
	cfg.OutboundConfigs = json_util.RawMessage(raw)
	return nil
}

func exitOutbound(exit model.TorExit) map[string]any {
	return map[string]any{
		"tag": OutboundTag(exit.Country), "protocol": "socks",
		"settings": map[string]any{"servers": []any{map[string]any{"address": "127.0.0.1", "port": exit.Port}}},
	}
}
