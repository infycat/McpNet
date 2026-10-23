package main

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPluginZipWithSingleWrapperFolder(t *testing.T) {
	path := makePluginZip(t, map[string]string{
		"shared-plugin/plugin.json": `{"id":"shared-plugin","name":"Shared plugin","enabled":true,"ui":{"title":"Tool","html":"<button>ok</button>"}}`,
		"shared-plugin/main.go":     "package main\nfunc OnWebSocket() {}\n",
	})

	pkg, err := readPluginZip(path)
	if err != nil {
		t.Fatalf("readPluginZip() error = %v", err)
	}
	if pkg.manifest.ID != "shared-plugin" || pkg.manifest.Enabled {
		t.Fatalf("unexpected imported manifest: %+v", pkg.manifest)
	}
	if pkg.manifest.UI == nil || pkg.manifest.UI.Title != "Tool" || pkg.manifest.UI.HTML != "<button>ok</button>" {
		t.Fatalf("custom UI was not preserved: %+v", pkg.manifest.UI)
	}
	if !strings.Contains(string(pkg.source), "OnWebSocket") {
		t.Fatalf("unexpected source: %q", pkg.source)
	}
}

func TestPluginUIValidationLimits(t *testing.T) {
	if err := validatePluginUI(&pluginUI{Title: "Tool", HTML: "<button>ok</button>"}); err != nil {
		t.Fatalf("valid plugin UI rejected: %v", err)
	}
	if err := validatePluginUI(&pluginUI{HTML: strings.Repeat("x", (256<<10)+1)}); err == nil {
		t.Fatal("expected oversized plugin UI to be rejected")
	}
	if err := validatePluginUI(&pluginUI{Title: strings.Repeat("x", 129)}); err == nil {
		t.Fatal("expected oversized plugin UI title to be rejected")
	}
}

func TestReadPluginZipRejectsFilesInDifferentFolders(t *testing.T) {
	path := makePluginZip(t, map[string]string{
		"plugin.json":   `{"id":"shared-plugin","name":"Shared plugin"}`,
		"other/main.go": "package main\n",
	})
	if _, err := readPluginZip(path); err == nil {
		t.Fatal("expected mismatched package layout to be rejected")
	}
}

func TestImportPluginPackageDoesNotOverwriteAndDefaultsDisabled(t *testing.T) {
	root := t.TempDir()
	manifest, err := parseImportedManifest([]byte(`{"id":"shared-plugin","name":"Shared plugin","enabled":true}`))
	if err != nil {
		t.Fatal(err)
	}
	pkg := pluginPackage{manifest: manifest, source: []byte("package main\n")}
	if err := installPluginPackageTo(root, pkg); err != nil {
		t.Fatalf("installPluginPackageTo() error = %v", err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(root, "shared-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var installed pluginManifest
	if err := json.Unmarshal(manifestBytes, &installed); err != nil {
		t.Fatal(err)
	}
	if installed.Enabled {
		t.Fatal("imported plugin should remain disabled")
	}
	if err := installPluginPackageTo(root, pkg); err == nil {
		t.Fatal("expected duplicate plugin ID to be rejected")
	}
}

func TestReadPluginFolderRequiresMainAndSingleJSON(t *testing.T) {
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "manifest.json"), []byte(`{"id":"shared-plugin","name":"Shared plugin"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPluginFolder(folder); err != nil {
		t.Fatalf("readPluginFolder() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(folder, "another.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPluginFolder(folder); err == nil {
		t.Fatal("expected multiple JSON files to be rejected")
	}
}

func makePluginZip(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plugin.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
