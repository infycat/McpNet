//go:build windows
// +build windows

package NFapi

import "C"
import (
	_ "embed"
	"github.com/qtgolang/McpNet/src/ProcessDrv/Info"
	. "github.com/qtgolang/McpNet/src/ProcessDrv/Info"
	"github.com/qtgolang/McpNet/src/Resource"
	"os"
	"path/filepath"
	"strings"
)

// 删除旧的驱动文件
func deleteOldFiles() {
	OldFileName := System32Dir + "\\drivers\\McpFilter.sys"
	//复制到临时目录去系统重启后才可删除
	_ = MoveFileToTempDir(OldFileName, "Mcp_"+RandomLetters(32)+extensionsTemp)
	// 只清理本程序以前留下的临时驱动文件，不再递归遍历整个临时目录。
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "Mcp_*"+extensionsTemp))
	for _, path := range matches {
		_ = os.Remove(path)
	}
}
// deleteOldFiles used to run from init(), walking the whole temp directory
// before the UI started (about 14 seconds on a normal machine). It is already
// called from Install(), which is only reached when the listener is started.

// System32Dir C:\Windows\system32\
var System32Dir = GetSystemDirectory()
var extensionsTemp = ".tmpSys"
var DriverFile = System32Dir + "\\drivers\\" + NF_DriverName + ".sys"

func UnInstall() {
	//复制到临时目录去系统重启后才可删除
	_ = MoveFileToTempDir(DriverFile, "Mcp_"+RandomLetters(32)+extensionsTemp)
	DrDLL := Info.WindowsDirectory + NF_DLLName + "64.dll"
	_ = MoveFileToTempDir(DrDLL, "Mcp_"+RandomLetters(32)+extensionsTemp)
	DrDLL = Info.WindowsDirectory + NF_DLLName + "32.dll"
	_ = MoveFileToTempDir(DrDLL, "Mcp_"+RandomLetters(32)+extensionsTemp)
}
func Install() string {
	deleteOldFiles()
	//XP直接打开不程序，所以就直接忽略
	s := []string{"OS", "Get", "Caption"}
	IsWin7 := strings.Index(Info.ExecCommand("Wmic", s), "Windows 7") != -1
	var oldValue uintptr
	if Info.Is64Windows {
		//如果是32位进程 禁止文件重定向 驱动只能写到 system32 目录
		if !WindowsX64 {
			oldValue = Info.Wow64DisableWow64FsRedirection()
		}
	}
	if !Info.Exists(DriverFile) {
		if IsWin7 {
			if Info.Is64Windows {
				Info.WriteFile(DriverFile, Resource.TdiAmd64Netfilter2)

			} else {
				Info.WriteFile(DriverFile, Resource.TdiI386Netfilter2)
			}
		} else {
			if Info.Is64Windows {
				Info.WriteFile(DriverFile, Resource.WfpAmd64Netfilter2)
			} else {
				Info.WriteFile(DriverFile, Resource.WfpI386Netfilter2)
			}
		}
	}
	if Info.Is64Windows {
		//如果是32位进程 恢复文件重定向
		if !WindowsX64 {
			Info.Wow64RevertWow64FsRedirection(oldValue)
		}
	}

	DrDLL := ""
	if WindowsX64 {
		DrDLL = Info.WindowsDirectory + NF_DLLName + "64.dll"
		Info.WriteFile(DrDLL, Resource.NfapiX64Nfapi)
	} else {
		DrDLL = Info.WindowsDirectory + NF_DLLName + "32.dll"
		Info.WriteFile(DrDLL, Resource.NfapiWin32Nfapi)
	}
	return DrDLL
}
