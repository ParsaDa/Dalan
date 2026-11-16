package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/json_util"
)

const maxNodeTunnels = 1000

// ListTunnels returns the tunnel ends stored on the node.
func (r *Remote) ListTunnels(ctx context.Context) ([]model.ServerTunnel, error) {
	env, err := r.do(ctx, http.MethodGet, "panel/api/tunnels/list", nil)
	if err != nil {
		return nil, err
	}
	list, err := json_util.DecodeCappedList[model.ServerTunnel](env.Obj, maxNodeTunnels)
	if err != nil {
		return nil, fmt.Errorf("decode tunnel list: %w", err)
	}
	return list, nil
}

// ImportTunnel makes the node store the dialling end of a tunnel from its setup link.
func (r *Remote) ImportTunnel(ctx context.Context, link, exitOutbound string) (*model.ServerTunnel, error) {
	env, err := r.do(ctx, http.MethodPost, "panel/api/tunnels/import",
		map[string]string{"link": link, "exitOutbound": exitOutbound})
	if err != nil {
		return nil, err
	}
	var stored model.ServerTunnel
	if err := json.Unmarshal(env.Obj, &stored); err != nil {
		return nil, fmt.Errorf("decode imported tunnel: %w", err)
	}
	return &stored, nil
}

// DeleteTunnel removes one tunnel end from the node by the id it has there.
func (r *Remote) DeleteTunnel(ctx context.Context, id int) error {
	_, err := r.do(ctx, http.MethodPost, fmt.Sprintf("panel/api/tunnels/del/%d", id), nil)
	return err
}
