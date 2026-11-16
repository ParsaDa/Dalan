// Package enrolment lets a node abroad join this panel by dialling in, on both sides of the exchange.
package enrolment

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/enrol"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/panel"
)

const (
	linkLifetime     = 30 * time.Minute
	tunnelNamePrefix = "node-"
	// mainPanelTokenPrefix names the API token a node issues to its main panel; a random suffix follows.
	mainPanelTokenPrefix = "dalan-main-panel-"
	joinTimeout          = 30 * time.Second
)

// ErrGone covers every reason a link cannot be redeemed, so a caller learns nothing about which applies.
var ErrGone = errors.New("this enrolment link is not valid: it was used, cancelled or has expired")

// The tunnel is named "node-<name>" and a tunnel name holds 31 characters.
var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,25}$`)

// Spec is what an operator chooses when inviting a node.
type Spec struct {
	Name       string `json:"name" example:"de1"`
	PanelURL   string `json:"panelUrl" example:"https://panel.example.com:2053/"`
	Address    string `json:"address" example:"203.0.113.10"`
	Port       int    `json:"port" example:"8443"`
	Security   string `json:"security" example:"reality"`
	ServerName string `json:"serverName" example:"www.example.com"`
	// TunnelId, when set, carries the node's management on that existing entry tunnel; the address
	// and security fields are then taken from it.
	TunnelId int `json:"tunnelId,omitempty" example:"0"`
	// WithTunnel also makes an ordinary entry tunnel named after the node, and the management
	// rides on it; the node's other end of it is stored when the node joins.
	WithTunnel bool `json:"withTunnel,omitempty" example:"true"`
}

// Invitation is a new enrolment with the link that is shown once and never stored.
type Invitation struct {
	Id        int    `json:"id" example:"1"`
	Name      string `json:"name" example:"de1"`
	ExpiresAt int64  `json:"expiresAt" example:"1700001800"`
	Link      string `json:"link" example:"dalan://enrol/eyJ2IjoxfQ"`
	Command   string `json:"command" example:"dalan enrol 'dalan://enrol/eyJ2IjoxfQ'"`
	// InstallCommand installs Dalan on a fresh server and joins this panel in one go.
	InstallCommand string `json:"installCommand" example:"XUI_ENROL='dalan://enrol/eyJ2IjoxfQ' bash <(curl -Ls https://raw.githubusercontent.com/ParsaDa/dalan/main/install.sh)"`
}

const installScriptURL = "https://raw.githubusercontent.com/ParsaDa/dalan/main/install.sh"

// Service runs enrolments: Create, Redeem and the clean-up on the main panel, Join on the node.
type Service struct {
	tunnels service.TunnelService
	nodes   service.NodeService
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Service) purgeExpired() {
	var expired []model.NodeEnrolment
	if err := database.GetDB().Where("used_at = 0 AND expires_at < ?", time.Now().Unix()).Find(&expired).Error; err != nil {
		logger.Warning("enrolments: listing expired links:", err)
		return
	}
	for i := range expired {
		if err := s.remove(&expired[i]); err != nil {
			logger.Warning("enrolments: removing expired link:", err)
		}
	}
}

func (s *Service) remove(row *model.NodeEnrolment) error {
	if err := s.tunnels.DeleteManagement(row.TunnelId); err != nil {
		return err
	}
	if row.DataTunnelId > 0 {
		if err := s.tunnels.DeleteNodeTunnel(row.DataTunnelId); err != nil {
			return err
		}
	}
	return database.GetDB().Delete(&model.NodeEnrolment{}, row.Id).Error
}

// List returns the enrolments that are still waiting or tie a node to its tunnel.
func (s *Service) List() ([]model.NodeEnrolment, error) {
	s.purgeExpired()
	var rows []model.NodeEnrolment
	err := database.GetDB().Order("id asc").Find(&rows).Error
	return rows, err
}

// Create opens a management tunnel for a node that has not joined yet and returns its one-time link.
func (s *Service) Create(spec Spec) (*Invitation, error) {
	s.purgeExpired()
	name := strings.TrimSpace(spec.Name)
	if !namePattern.MatchString(name) {
		return nil, fmt.Errorf("node name %q must be lower-case letters, digits and dashes, at most 26 characters", spec.Name)
	}
	db := database.GetDB()
	var taken int64
	if err := db.Model(&model.Node{}).Where("name = ?", name).Count(&taken).Error; err != nil {
		return nil, err
	}
	if taken == 0 {
		if err := db.Model(&model.NodeEnrolment{}).Where("name = ?", name).Count(&taken).Error; err != nil {
			return nil, err
		}
	}
	if taken > 0 {
		return nil, fmt.Errorf("a node or enrolment named %q already exists", name)
	}
	offer, err := enrol.NewOffer(spec.PanelURL)
	if err != nil {
		return nil, err
	}
	if spec.WithTunnel && spec.TunnelId > 0 {
		return nil, errors.New("the node already rides on an existing tunnel; it does not need one of its own")
	}
	var portal, data *model.ServerTunnel
	switch {
	case spec.TunnelId > 0:
		portal, err = s.tunnels.CreateManagementRider(spec.TunnelId, tunnelNamePrefix+name)
	case spec.WithTunnel:
		portal, data, err = s.createTunnelWithChannel(name, spec)
	default:
		portal, err = s.tunnels.CreateManagementPortal(tunnel.PortalSpec{
			Name: tunnelNamePrefix + name, Address: spec.Address, Port: spec.Port,
			Security: spec.Security, ServerName: spec.ServerName,
		})
	}
	if err != nil {
		return nil, err
	}
	row := model.NodeEnrolment{
		Name: name, PublicId: offer.ID, TokenHash: hashToken(offer.Token), PrivateKey: offer.PrivateKey,
		ExpiresAt: time.Now().Add(linkLifetime).Unix(), TunnelId: portal.Id,
	}
	if data != nil {
		row.DataTunnelId = data.Id
	}
	if err := db.Create(&row).Error; err != nil {
		if delErr := s.remove(&row); delErr != nil {
			logger.Warning("enrolments: removing the tunnel of a failed enrolment:", delErr)
		}
		return nil, err
	}
	return &Invitation{
		Id: row.Id, Name: row.Name, ExpiresAt: row.ExpiresAt,
		Link: offer.Link, Command: "dalan enrol '" + offer.Link + "'",
		InstallCommand: "XUI_ENROL='" + offer.Link + "' bash <(curl -Ls " + installScriptURL + ")",
	}, nil
}

// createTunnelWithChannel makes the node's ordinary entry tunnel and its management channel on the same port.
func (s *Service) createTunnelWithChannel(name string, spec Spec) (channel, data *model.ServerTunnel, err error) {
	data, err = s.tunnels.CreatePortal(tunnel.PortalSpec{
		Name: name, Address: spec.Address, Port: spec.Port, Security: spec.Security, ServerName: spec.ServerName,
	}, nil)
	if err != nil {
		return nil, nil, err
	}
	channel, err = s.tunnels.CreateManagementRider(data.Id, tunnelNamePrefix+name)
	if err != nil {
		if delErr := s.tunnels.DeleteNodeTunnel(data.Id); delErr != nil {
			logger.Warning("enrolments: removing the tunnel of a failed enrolment:", delErr)
		}
		return nil, nil, err
	}
	return channel, data, nil
}

// Delete cancels a waiting enrolment and closes its tunnel; a joined node is removed from the node list instead.
func (s *Service) Delete(id int) error {
	var row model.NodeEnrolment
	if err := database.GetDB().First(&row, id).Error; err != nil {
		return fmt.Errorf("enrolment %d does not exist", id)
	}
	if row.NodeId != 0 {
		return errors.New("this enrolment belongs to a node; delete the node instead")
	}
	return s.remove(&row)
}

// NodeDeleted closes the management tunnel of a node that was removed from the node list.
func (s *Service) NodeDeleted(nodeID int) error {
	var rows []model.NodeEnrolment
	if err := database.GetDB().Where("node_id = ?", nodeID).Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		if err := s.remove(&rows[i]); err != nil {
			return err
		}
	}
	return nil
}

func validNodeInfo(info enrol.NodeInfo) bool {
	return info.Port >= 1 && info.Port <= 65535 && info.APIToken != "" &&
		(info.Scheme == "http" || info.Scheme == "https")
}

// Redeem answers a node's sealed request: it registers the node behind its management port and returns the tunnel.
// remoteIP is where the request came from; it becomes the node's public address for client links.
func (s *Service) Redeem(body []byte, remoteIP string) ([]byte, error) {
	s.purgeExpired()
	id, err := enrol.RequestID(body)
	if err != nil {
		return nil, ErrGone
	}
	db := database.GetDB()
	var rows []model.NodeEnrolment
	if err := db.Where("public_id = ? AND used_at = 0", id).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrGone
	}
	row := rows[0]
	info, sealReply, err := enrol.OpenRequest(row.PrivateKey, body)
	if err != nil {
		return nil, ErrGone
	}
	if subtle.ConstantTimeCompare([]byte(hashToken(info.Token)), []byte(row.TokenHash)) != 1 || !validNodeInfo(info) {
		return nil, ErrGone
	}
	var portal model.ServerTunnel
	if err := db.First(&portal, row.TunnelId).Error; err != nil {
		return nil, ErrGone
	}
	// A BackPack tunnel is already set up on the node, so the channel is a forward on it, not a tunnel to import.
	onBackpack := portal.ParentId > 0 && portal.Engine == model.TunnelEngineBackpack
	var setup string
	if !onBackpack {
		if setup, err = tunnel.SetupLink(portal); err != nil {
			return nil, err
		}
	}

	// The conditional update is the single point where a link is spent; a concurrent second request loses here.
	claim := db.Model(&model.NodeEnrolment{}).Where("id = ? AND used_at = 0", row.Id).
		Updates(map[string]any{"used_at": time.Now().Unix(), "token_hash": "", "private_key": ""})
	if claim.Error != nil {
		return nil, claim.Error
	}
	if claim.RowsAffected != 1 {
		return nil, ErrGone
	}
	node := &model.Node{
		Name: row.Name, Scheme: info.Scheme, Address: "127.0.0.1", PublicAddress: remoteIP, Port: portal.ManagePort,
		BasePath: info.BasePath, ApiToken: info.APIToken, Enable: true, AllowPrivateAddress: true,
		// The hop is loopback to loopback inside the tunnel, and the node's certificate names its public host.
		TlsVerifyMode: "skip",
	}
	if err := s.nodes.Create(node); err != nil {
		if delErr := s.remove(&row); delErr != nil {
			logger.Warning("enrolments: removing an enrolment whose node could not be stored:", delErr)
		}
		return nil, err
	}
	if err := db.Model(&model.NodeEnrolment{}).Where("id = ?", row.Id).Update("node_id", node.Id).Error; err != nil {
		return nil, err
	}
	if onBackpack {
		if err := db.Model(&model.ServerTunnel{}).Where("id = ?", portal.Id).Update("manage_target", info.Port).Error; err != nil {
			return nil, err
		}
		if err := s.tunnels.ReconcileEngines(); err != nil {
			logger.Warning("enrolments: applying the node's forward to its BackPack tunnel:", err)
		}
	}
	grant := enrol.Grant{Tunnel: setup}
	if row.DataTunnelId > 0 {
		if grant.DataTunnel, err = s.grantDataTunnel(row.DataTunnelId, node.Id); err != nil {
			return nil, err
		}
	}
	return sealReply(grant)
}

// grantDataTunnel ties the node's traffic tunnel to the node, so deleting it reaches the node's end too.
func (s *Service) grantDataTunnel(tunnelID, nodeID int) (string, error) {
	var data model.ServerTunnel
	if err := database.GetDB().First(&data, tunnelID).Error; err != nil {
		return "", err
	}
	if err := database.GetDB().Model(&model.ServerTunnel{}).Where("id = ?", tunnelID).Update("node_id", nodeID).Error; err != nil {
		return "", err
	}
	return tunnel.SetupLink(data)
}

func loopbackOrAny(listen string) bool {
	if listen == "" {
		return true
	}
	ip := net.ParseIP(listen)
	return ip != nil && (ip.IsUnspecified() || ip.IsLoopback())
}

// Join runs on the node: it hands this panel's details to the main panel in the link and stores the tunnel it gets back.
func (s *Service) Join(ctx context.Context, rawLink string) error {
	link, err := enrol.ParseLink(rawLink)
	if err != nil {
		return err
	}
	settings := service.SettingService{}
	port, err := settings.GetPort()
	if err != nil {
		return err
	}
	basePath, err := settings.GetBasePath()
	if err != nil {
		return err
	}
	listen, err := settings.GetListen()
	if err != nil {
		return err
	}
	if !loopbackOrAny(listen) {
		return fmt.Errorf("this panel listens only on %s; the management tunnel delivers to 127.0.0.1, so clear the listen IP first", listen)
	}
	certFile, _ := settings.GetCertFile()
	keyFile, _ := settings.GetKeyFile()
	scheme := "http"
	if certFile != "" && keyFile != "" {
		scheme = "https"
	}

	tokens := panel.ApiTokenService{}
	// A token of its own per attempt, so a refused join cannot disturb the channel that works now.
	attempt := make([]byte, 6)
	if _, err := rand.Read(attempt); err != nil {
		return err
	}
	token, err := tokens.Create(mainPanelTokenPrefix+hex.EncodeToString(attempt), model.ApiScopeNodeAdmin, 0)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, joinTimeout)
	defer cancel()
	// The exchange is sealed with the key in the link, which is what authenticates the main panel; its
	// certificate is often self-signed or absent.
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // authenticated by the link key
	}}
	grant, err := enrol.Redeem(ctx, client, link, enrol.NodeInfo{
		Token: link.Token, Port: port, BasePath: basePath, Scheme: scheme, APIToken: token.Token,
	})
	if err == nil && grant.Tunnel != "" {
		_, err = s.tunnels.ImportManagementBridge(grant.Tunnel, port)
	}
	if err == nil && grant.DataTunnel != "" {
		_, err = s.tunnels.Import(grant.DataTunnel, "")
	}
	if err != nil {
		if delErr := tokens.Delete(token.Id); delErr != nil {
			logger.Warning("enrolments: removing the token of a failed join:", delErr)
		}
		return err
	}
	return revokeOtherMainPanelTokens(tokens, token.Id)
}

// A node answers to one main panel: once a join succeeds, the tokens of earlier main panels stop working.
func revokeOtherMainPanelTokens(tokens panel.ApiTokenService, keep int) error {
	all, err := tokens.List()
	if err != nil {
		return err
	}
	for _, t := range all {
		if t.Id != keep && strings.HasPrefix(t.Name, mainPanelTokenPrefix) {
			if err := tokens.Delete(t.Id); err != nil {
				return err
			}
		}
	}
	return nil
}
