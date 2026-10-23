package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	mcpPermissionConfirm      = "confirm"
	mcpPermissionCustom       = "custom"
	mcpPermissionUnrestricted = "unrestricted"
)

// mcpAccessSettings are local, persistent grants controlled from the MCP
// settings panel. The zero-value policy remains confirmation-per-operation.
type mcpAccessSettings struct {
	Mode               string `json:"mode"`
	AllowPrivateRead   bool   `json:"allow_private_read"`
	AllowSettings      bool   `json:"allow_settings"`
	AllowPluginCode    bool   `json:"allow_plugin_code"`
	AllowNetworkWrites bool   `json:"allow_network_writes"`
}

func mcpAccessSettingsPath() string {
	return filepath.Join(mcpDataDir(), "mcp-access.json")
}

func normalizeMCPAccessSettings(settings mcpAccessSettings) mcpAccessSettings {
	switch strings.ToLower(strings.TrimSpace(settings.Mode)) {
	case mcpPermissionCustom:
		settings.Mode = mcpPermissionCustom
	case mcpPermissionUnrestricted:
		settings.Mode = mcpPermissionUnrestricted
	default:
		settings.Mode = mcpPermissionConfirm
	}
	return settings
}

func loadMCPAccessSettings() mcpAccessSettings {
	data, err := os.ReadFile(mcpAccessSettingsPath())
	if err != nil {
		return mcpAccessSettings{Mode: mcpPermissionConfirm}
	}
	var settings mcpAccessSettings
	if json.Unmarshal(data, &settings) != nil {
		return mcpAccessSettings{Mode: mcpPermissionConfirm}
	}
	return normalizeMCPAccessSettings(settings)
}

func saveMCPAccessSettings(settings mcpAccessSettings) error {
	settings = normalizeMCPAccessSettings(settings)
	if err := os.MkdirAll(mcpDataDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	path := mcpAccessSettingsPath()
	return os.WriteFile(path, data, 0600)
}

func (s mcpAccessSettings) permits(category string) bool {
	s = normalizeMCPAccessSettings(s)
	if s.Mode == mcpPermissionUnrestricted {
		return true
	}
	if s.Mode != mcpPermissionCustom {
		return false
	}
	switch category {
	case "private_read":
		return s.AllowPrivateRead
	case "settings":
		return s.AllowSettings
	case "plugin_code":
		return s.AllowPluginCode
	case "network_write":
		return s.AllowNetworkWrites
	default:
		return false
	}
}

func (a *App) mcpAccessInfo() mcpAccessSettings {
	return loadMCPAccessSettings()
}

func (a *App) setMCPAccessSettings(args map[string]any) (mcpAccessSettings, error) {
	settings := mcpAccessSettings{
		Mode:               stringArg(args, "Mode"),
		AllowPrivateRead:   boolArg(args, "AllowPrivateRead"),
		AllowSettings:      boolArg(args, "AllowSettings"),
		AllowPluginCode:    boolArg(args, "AllowPluginCode"),
		AllowNetworkWrites: boolArg(args, "AllowNetworkWrites"),
	}
	settings = normalizeMCPAccessSettings(settings)
	if err := saveMCPAccessSettings(settings); err != nil {
		return loadMCPAccessSettings(), err
	}
	return settings, nil
}

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func boolArg(args map[string]any, key string) bool {
	value, _ := args[key].(bool)
	return value
}
