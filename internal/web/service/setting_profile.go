package service

import (
	"net"
	"regexp"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/util/common"
)

// ServerProfile is the part of the settings the first-run wizard and the accent picker change on
// their own, so neither has to send (and re-validate) every other setting. A nil field is left as it is.
type ServerProfile struct {
	ServerName    *string `json:"serverName" form:"serverName"`
	ServerCountry *string `json:"serverCountry" form:"serverCountry"`
	ServerAddress *string `json:"serverAddress" form:"serverAddress"`
	Accent        *string `json:"accent" form:"accent"`
	SetupDone     *bool   `json:"setupDone" form:"setupDone"`
}

const (
	maxServerNameLength = 80
	// accentSettingKey is written only by UpdateServerProfile, never by the whole-settings save.
	accentSettingKey = "accent"
)

var (
	countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)
	accentIDPattern    = regexp.MustCompile(`^[a-z]{2,16}$`)
	hostLabelPattern   = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
)

// validServerAddress accepts an IP address or a bare host name; a scheme, port or path is not an address.
func validServerAddress(v string) bool {
	if v == "" {
		return true
	}
	if net.ParseIP(strings.Trim(v, "[]")) != nil {
		return true
	}
	if len(v) > 253 {
		return false
	}
	for _, label := range strings.Split(v, ".") {
		if !hostLabelPattern.MatchString(label) {
			return false
		}
	}
	return true
}

// UpdateServerProfile checks every field it was given and only then writes them, so a bad value saves nothing.
func (s *SettingService) UpdateServerProfile(p ServerProfile) error {
	if p.ServerName != nil {
		name := strings.TrimSpace(*p.ServerName)
		if len([]rune(name)) > maxServerNameLength {
			return common.NewError("server name is longer than 80 characters")
		}
		p.ServerName = &name
	}
	if p.ServerCountry != nil && *p.ServerCountry != "" && !countryCodePattern.MatchString(*p.ServerCountry) {
		return common.NewError("server country must be a two-letter code such as DE:", *p.ServerCountry)
	}
	if p.ServerAddress != nil {
		address := strings.TrimSpace(*p.ServerAddress)
		if !validServerAddress(address) {
			return common.NewError("server address must be an IP address or a domain without http:// or a path:", address)
		}
		p.ServerAddress = &address
	}
	if p.Accent != nil && *p.Accent != "" && !accentIDPattern.MatchString(*p.Accent) {
		return common.NewError("accent is not a valid colour name:", *p.Accent)
	}

	for key, value := range map[string]*string{
		"serverName":     p.ServerName,
		"serverCountry":  p.ServerCountry,
		"serverAddress":  p.ServerAddress,
		accentSettingKey: p.Accent,
	} {
		if value == nil {
			continue
		}
		if err := s.setString(key, *value); err != nil {
			return err
		}
	}
	if p.SetupDone != nil {
		return s.setBool("setupDone", *p.SetupDone)
	}
	return nil
}

// GetServerAddress returns the address people connect to this server by, or "" when none was set.
func (s *SettingService) GetServerAddress() (string, error) {
	return s.getString("serverAddress")
}
