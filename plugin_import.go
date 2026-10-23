package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxPluginPackageFileSize = 4 << 20
const maxPluginPackageSize = 16 << 20

type pluginPackage struct {
	manifest pluginManifest
	source   []byte
}

func readPluginFolder(folder string) (pluginPackage, error) {
	var pkg pluginPackage
	info, err := os.Stat(folder)
	if err != nil {
		return pkg, fmt.Errorf("讀取插件資料夾失敗：%w", err)
	}
	if !info.IsDir() {
		return pkg, fmt.Errorf("所選路徑不是資料夾")
	}

	entries, err := os.ReadDir(folder)
	if err != nil {
		return pkg, fmt.Errorf("讀取插件資料夾失敗：%w", err)
	}
	manifestNames := make([]string, 0, 1)
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			manifestNames = append(manifestNames, entry.Name())
		}
	}
	if len(manifestNames) != 1 {
		return pkg, fmt.Errorf("插件資料夾必須在根目錄包含且只包含一個 JSON 清單檔")
	}

	manifestBytes, err := readLimitedFile(filepath.Join(folder, manifestNames[0]), maxPluginPackageFileSize)
	if err != nil {
		return pkg, fmt.Errorf("讀取插件 JSON 失敗：%w", err)
	}
	pkg.manifest, err = parseImportedManifest(manifestBytes)
	if err != nil {
		return pkg, err
	}
	pkg.source, err = readLimitedFile(filepath.Join(folder, "main.go"), maxPluginPackageFileSize)
	if err != nil {
		return pkg, fmt.Errorf("讀取 main.go 失敗：%w", err)
	}
	if len(strings.TrimSpace(string(pkg.source))) == 0 {
		return pluginPackage{}, fmt.Errorf("main.go 不能為空")
	}
	return pkg, nil
}

func readPluginZip(zipPath string) (pluginPackage, error) {
	var pkg pluginPackage
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return pkg, fmt.Errorf("無法開啟 ZIP：%w", err)
	}
	defer archive.Close()

	if len(archive.File) > 512 {
		return pkg, fmt.Errorf("ZIP 內檔案數量過多")
	}
	var totalSize uint64
	var manifestFile, sourceFile *zip.File
	manifestDir, sourceDir := "", ""
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > maxPluginPackageSize-totalSize {
			return pkg, fmt.Errorf("ZIP 解壓後的檔案總量不可超過 16 MB")
		}
		totalSize += file.UncompressedSize64
		if totalSize > maxPluginPackageSize {
			return pkg, fmt.Errorf("ZIP 解壓後的檔案總量不可超過 16 MB")
		}
		if file.UncompressedSize64 > maxPluginPackageFileSize {
			return pkg, fmt.Errorf("ZIP 中單一檔案不可超過 4 MB：%s", filepath.Base(file.Name))
		}

		name := strings.ReplaceAll(file.Name, "\\", "/")
		if strings.HasPrefix(name, "/") || filepath.IsAbs(file.Name) {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
		if clean == ".." || strings.HasPrefix(clean, "../") {
			continue
		}
		parts := strings.Split(clean, "/")
		if len(parts) > 2 || len(parts) == 0 {
			continue
		}
		parent := ""
		base := parts[len(parts)-1]
		if len(parts) == 2 {
			parent = parts[0]
			if parent == "" || parent == "." || parent == ".." {
				continue
			}
		}
		if base == "main.go" {
			if sourceFile != nil {
				return pkg, fmt.Errorf("ZIP 中找到多個 main.go")
			}
			if file.Mode()&os.ModeSymlink != 0 {
				return pkg, fmt.Errorf("main.go 不可為符號連結")
			}
			sourceFile, sourceDir = file, parent
		} else if strings.EqualFold(filepath.Ext(base), ".json") {
			if manifestFile != nil {
				return pkg, fmt.Errorf("ZIP 中只能有一個插件 JSON 清單")
			}
			if file.Mode()&os.ModeSymlink != 0 {
				return pkg, fmt.Errorf("插件 JSON 不可為符號連結")
			}
			manifestFile, manifestDir = file, parent
		}
	}
	if manifestFile == nil || sourceFile == nil {
		return pkg, fmt.Errorf("ZIP 必須包含同一層目錄中的 main.go 和一個 JSON 清單")
	}
	if manifestDir != sourceDir {
		return pkg, fmt.Errorf("main.go 和 JSON 清單必須位於 ZIP 根目錄或同一個插件資料夾內")
	}

	manifestBytes, err := readZipEntry(manifestFile)
	if err != nil {
		return pkg, fmt.Errorf("讀取插件 JSON 失敗：%w", err)
	}
	pkg.manifest, err = parseImportedManifest(manifestBytes)
	if err != nil {
		return pkg, err
	}
	pkg.source, err = readZipEntry(sourceFile)
	if err != nil {
		return pkg, fmt.Errorf("讀取 main.go 失敗：%w", err)
	}
	if len(strings.TrimSpace(string(pkg.source))) == 0 {
		return pluginPackage{}, fmt.Errorf("main.go 不能為空")
	}
	return pkg, nil
}

func readZipEntry(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxPluginPackageFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPluginPackageFileSize {
		return nil, fmt.Errorf("ZIP 內檔案超過 4 MB 限制")
	}
	return data, nil
}

func readLimitedFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("只接受一般檔案，不接受連結或特殊檔案")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("檔案超過 %d MB 限制", limit>>20)
	}
	return data, nil
}

func parseImportedManifest(data []byte) (pluginManifest, error) {
	var manifest pluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("插件 JSON 格式錯誤：%w", err)
	}
	if strings.TrimSpace(manifest.ID) == "" {
		return manifest, fmt.Errorf("插件 JSON 缺少 id")
	}
	if sanitizePluginID(manifest.ID) != manifest.ID || manifest.ID == "." || manifest.ID == ".." {
		return manifest, fmt.Errorf("插件 id 只能使用不含路徑符號的名稱")
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return manifest, fmt.Errorf("插件 JSON 缺少 name")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		manifest.Version = "1.0.0"
	}
	if err := validatePluginUI(manifest.UI); err != nil {
		return pluginManifest{}, err
	}
	// Imported code must never run until the user explicitly enables it.
	manifest.Enabled = false
	return manifest, nil
}

func installPluginPackage(pkg pluginPackage) error {
	hookMu.Lock()
	defer hookMu.Unlock()

	if err := ensurePluginDir(); err != nil {
		return fmt.Errorf("建立插件資料夾失敗：%w", err)
	}
	if err := installPluginPackageTo(pluginDataDir(), pkg); err != nil {
		return err
	}
	invalidatePluginCache()
	return nil
}

func installPluginPackageTo(root string, pkg pluginPackage) error {
	if err := os.MkdirAll(root, 0700); err != nil {
		return fmt.Errorf("建立插件資料夾失敗：%w", err)
	}
	target := filepath.Join(root, sanitizePluginID(pkg.manifest.ID))
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("插件 ID「%s」已存在；未覆蓋任何檔案", pkg.manifest.ID)
	} else if !os.IsNotExist(err) {
		return err
	}

	tempDir, err := os.MkdirTemp(root, ".plugin-import-*")
	if err != nil {
		return fmt.Errorf("建立暫存插件目錄失敗：%w", err)
	}
	defer os.RemoveAll(tempDir)
	manifestBytes, err := json.MarshalIndent(pkg.manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(pluginManifestPath(tempDir), manifestBytes, 0600); err != nil {
		return err
	}
	if err := os.WriteFile(pluginSourcePath(tempDir), pkg.source, 0600); err != nil {
		return err
	}
	if err := os.Rename(tempDir, target); err != nil {
		return fmt.Errorf("安裝插件失敗：%w", err)
	}
	return nil
}

func importPluginFromFolder(folder string) error {
	pkg, err := readPluginFolder(folder)
	if err != nil {
		return err
	}
	return installPluginPackage(pkg)
}

func importPluginFromZip(zipPath string) error {
	pkg, err := readPluginZip(zipPath)
	if err != nil {
		return err
	}
	return installPluginPackage(pkg)
}
