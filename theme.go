package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type themeSettings struct {
	Key string `json:"key"`
}

var (
	themeMu    sync.Mutex
	themeCache = themeSettings{Key: "shine"}
	themePath  = ""
)

func themeSettingsPath() string {
	themeMu.Lock()
	defer themeMu.Unlock()
	if themePath == "" {
		themePath = filepath.Join(mcpDataDir(), "theme.json")
	}
	return themePath
}

func loadThemeSettings() themeSettings {
	themeMu.Lock()
	defer themeMu.Unlock()
	if themePath == "" {
		themePath = filepath.Join(mcpDataDir(), "theme.json")
	}
	bs, err := os.ReadFile(themePath)
	if err != nil {
		return themeCache
	}
	var t themeSettings
	if json.Unmarshal(bs, &t) != nil || t.Key == "" {
		return themeCache
	}
	themeCache = t
	return themeCache
}

func saveThemeSettings(t themeSettings) error {
	if t.Key == "" {
		t.Key = "shine"
	}
	themeMu.Lock()
	defer themeMu.Unlock()
	if themePath == "" {
		themePath = filepath.Join(mcpDataDir(), "theme.json")
	}
	if err := os.MkdirAll(mcpDataDir(), 0700); err != nil {
		return err
	}
	bs, _ := json.MarshalIndent(t, "", "  ")
	if err := os.WriteFile(themePath, bs, 0600); err != nil {
		return err
	}
	themeCache = t
	return nil
}
