package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
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

// DialPorts makes the node open a TCP connection to each port of host and report how it went.
func (r *Remote) DialPorts(ctx context.Context, host string, ports []int) ([]tunnel.PortResult, error) {
	env, err := r.do(ctx, http.MethodPost, "panel/api/tunnels/dial", map[string]any{"host": host, "ports": ports})
	if err != nil {
		return nil, err
	}
	list, err := json_util.DecodeCappedList[tunnel.PortResult](env.Obj, tunnel.MaxRouteTestPorts)
	if err != nil {
		return nil, fmt.Errorf("decode dial results: %w", err)
	}
	return list, nil
}

// FetchEngineRelease makes the node download a release of one engine (empty version: the newest) and
// fetches it in parts, since one call may not run longer than the request timeout.
func (r *Remote) FetchEngineRelease(ctx context.Context, engineID, version string) (engine.Release, error) {
	var rel engine.Release
	for {
		chunk, err := r.engineChunk(ctx, engineID, version, len(rel.Archive))
		if err != nil {
			return engine.Release{}, err
		}
		if chunk.Pending {
			select {
			case <-ctx.Done():
				return engine.Release{}, ctx.Err()
			case <-time.After(engineChunkPoll):
			}
			continue
		}
		if len(rel.Archive) == 0 {
			rel.Version, rel.Sums = chunk.Version, chunk.Sums
		}
		if chunk.Size > maxEngineArchive || len(rel.Archive)+len(chunk.Data) > chunk.Size {
			return engine.Release{}, errors.New("the node sent a release larger than it announced")
		}
		rel.Archive = append(rel.Archive, chunk.Data...)
		if len(rel.Archive) == chunk.Size {
			return rel, nil
		}
		if len(chunk.Data) == 0 {
			return engine.Release{}, errors.New("the node stopped sending the release part way")
		}
	}
}

// ListEngineReleases asks the node for the newest releases of one engine, read from where the node is.
func (r *Remote) ListEngineReleases(ctx context.Context, engineID string) ([]string, error) {
	env, err := r.do(ctx, http.MethodPost, "panel/api/tunnels/engine/releases", map[string]any{"engine": engineID})
	if err != nil {
		return nil, err
	}
	var versions []string
	if err := json.Unmarshal(env.Obj, &versions); err != nil {
		return nil, fmt.Errorf("decode engine releases: %w", err)
	}
	return versions, nil
}

const (
	engineChunkPoll  = time.Second
	maxEngineArchive = 200 << 20
)

func (r *Remote) engineChunk(ctx context.Context, engineID, version string, offset int) (engine.Chunk, error) {
	env, err := r.do(ctx, http.MethodPost, "panel/api/tunnels/engine/release",
		map[string]any{"engine": engineID, "version": version, "offset": offset})
	if err != nil {
		return engine.Chunk{}, err
	}
	var chunk engine.Chunk
	if err := json.Unmarshal(env.Obj, &chunk); err != nil {
		return engine.Chunk{}, fmt.Errorf("decode engine release: %w", err)
	}
	return chunk, nil
}
