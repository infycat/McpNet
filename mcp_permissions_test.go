package main

import "testing"

func TestMCPAccessSettingsRequireConfirmationByDefault(t *testing.T) {
	settings := normalizeMCPAccessSettings(mcpAccessSettings{})
	if settings.Mode != mcpPermissionConfirm {
		t.Fatalf("default mode = %q, want %q", settings.Mode, mcpPermissionConfirm)
	}
	for _, category := range []string{"private_read", "settings", "plugin_code", "network_write", "generic"} {
		if settings.permits(category) {
			t.Errorf("default policy unexpectedly permits %q", category)
		}
	}
}

func TestMCPAccessSettingsCustomCategoriesAreIndependent(t *testing.T) {
	settings := mcpAccessSettings{Mode: mcpPermissionCustom, AllowPrivateRead: true, AllowPluginCode: true}
	for category, want := range map[string]bool{
		"private_read":  true,
		"plugin_code":   true,
		"settings":      false,
		"network_write": false,
		"generic":       false,
	} {
		if got := settings.permits(category); got != want {
			t.Errorf("permits(%q) = %t, want %t", category, got, want)
		}
	}
}

func TestMCPAccessSettingsUnrestrictedPermitsAllCategories(t *testing.T) {
	settings := mcpAccessSettings{Mode: mcpPermissionUnrestricted}
	for _, category := range []string{"private_read", "settings", "plugin_code", "network_write", "generic", "future_category"} {
		if !settings.permits(category) {
			t.Errorf("unrestricted policy does not permit %q", category)
		}
	}
}

func TestMCPAccessSettingsUnknownModeFallsBackToConfirmation(t *testing.T) {
	settings := normalizeMCPAccessSettings(mcpAccessSettings{Mode: "unexpected", AllowPrivateRead: true})
	if settings.Mode != mcpPermissionConfirm || settings.permits("private_read") {
		t.Fatalf("unknown mode did not fail closed: %+v", settings)
	}
}
