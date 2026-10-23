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
	manifestJSON, err := json.Marshal(pkg.manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(manifestJSON), `"ui"`) {
		t.Fatalf("legacy manifest UI should be ignored; windows are defined by Go hooks: %s", manifestJSON)
	}
	if !strings.Contains(string(pkg.source), "OnWebSocket") {
		t.Fatalf("unexpected source: %q", pkg.source)
	}

	archiveBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readPluginZipBytes(archiveBytes); err != nil {
		t.Fatalf("readPluginZipBytes() error = %v", err)
	}
}

func TestReadPluginZipBytesRejectsOversizedArchive(t *testing.T) {
	if _, err := readPluginZipBytes(make([]byte, maxPluginPackageSize+1)); err == nil {
		t.Fatal("expected oversized ZIP bytes to be rejected")
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
