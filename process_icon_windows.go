//go:build windows

package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"os"

	"changeme/CommAnd"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

)

const (
	processQueryLimitedInformation = 0x1000
	shgfiIcon                      = 0x000000100
	shgfiSmallIcon                 = 0x000000001
	dibRGBColors                   = 0
)

type shFileInfo struct {
	hIcon         syscall.Handle
	iIcon         int32
	dwAttributes  uint32
	szDisplayName [260]uint16
	szTypeName    [80]uint16
}

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  syscall.Handle
	hbmColor syscall.Handle
}

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

var (
	user32DLL   = syscall.NewLazyDLL("user32.dll")
	gdi32DLL    = syscall.NewLazyDLL("gdi32.dll")
	shell32DLL  = syscall.NewLazyDLL("shell32.dll")
	kernel32DLL = syscall.NewLazyDLL("kernel32.dll")

	procOpenProcess                 = kernel32DLL.NewProc("OpenProcess")
	procQueryFullProcessImageNameW  = kernel32DLL.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                 = kernel32DLL.NewProc("CloseHandle")
	procSHGetFileInfoW              = shell32DLL.NewProc("SHGetFileInfoW")
	procGetDC                       = user32DLL.NewProc("GetDC")
	procReleaseDC                   = user32DLL.NewProc("ReleaseDC")
	procDestroyIcon                 = user32DLL.NewProc("DestroyIcon")
	procEnumWindows              = user32DLL.NewProc("EnumWindows")
	procIsWindowVisible          = user32DLL.NewProc("IsWindowVisible")
	procGetWindowTextW           = user32DLL.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32DLL.NewProc("GetWindowThreadProcessId")
	procGetWindowLongW           = user32DLL.NewProc("GetWindowLongW")
	procGetWindowLongPtrW        = user32DLL.NewProc("GetWindowLongPtrW")
	procGetObjectW                  = gdi32DLL.NewProc("GetObjectW")
	procGetDIBits                   = gdi32DLL.NewProc("GetDIBits")
	procDeleteObject                = gdi32DLL.NewProc("DeleteObject")

	iconCacheMu sync.Mutex
	iconCache   = map[string]string{}
)

func processImagePath(pid int) string {
	h, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer procCloseHandle.Call(h)
	buf := make([]uint16, 1024)
	n := uint32(len(buf))
	r, _, _ := procQueryFullProcessImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if r == 0 || n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}

// genericIcon returns the shell's default executable icon. Protected processes
// such as System/Secure System cannot be opened, so their image path is empty;
// Task Manager shows a generic icon for them, and so do we.
func genericIcon() string {
	iconCacheMu.Lock()
	if v, ok := iconCache["<generic>"]; ok {
		iconCacheMu.Unlock()
		return v
	}
	iconCacheMu.Unlock()
	data := extractIconFromPath(`C:\Windows\explorer.exe`)
	if data == "" {
		return ""
	}
	iconCacheMu.Lock()
	iconCache["<generic>"] = data
	iconCacheMu.Unlock()
	return data
}

func extractIconFromPath(path string) string {
	if path == "" {
		return ""
	}
	iconCacheMu.Lock()
	if v, ok := iconCache[path]; ok {
		iconCacheMu.Unlock()
		return v
	}
	iconCacheMu.Unlock()

	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	var info shFileInfo
	r, _, _ := procSHGetFileInfoW.Call(
		uintptr(unsafe.Pointer(ptr)), 0,
		uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info),
		shgfiIcon|shgfiSmallIcon,
	)
	if r == 0 || info.hIcon == 0 {
		return ""
	}
	defer procDestroyIcon.Call(uintptr(info.hIcon))

	var ii iconInfo
	if r, _, _ := procGetIconInfo.Call(uintptr(info.hIcon), uintptr(unsafe.Pointer(&ii))); r == 0 {
		return ""
	}
	if ii.hbmMask != 0 {
		defer procDeleteObject.Call(uintptr(ii.hbmMask))
	}
	if ii.hbmColor == 0 {
		return ""
	}
	defer procDeleteObject.Call(uintptr(ii.hbmColor))

	var bmp struct {
		bmType       int32
		bmWidth      int32
		bmHeight     int32
		bmWidthBytes int32
		bmPlanes     uint16
		bmBitsPixel  uint16
		bmBits       uintptr
	}
	if r, _, _ := procGetObjectW.Call(uintptr(ii.hbmColor), unsafe.Sizeof(bmp), uintptr(unsafe.Pointer(&bmp))); r == 0 {
		return ""
	}
	w, h := int(bmp.bmWidth), int(bmp.bmHeight)
	if w <= 0 || h <= 0 || w > 256 || h > 256 {
		return ""
	}
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return ""
	}
	defer procReleaseDC.Call(0, hdc)

	bih := bitmapInfoHeader{
		biSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		biWidth:       int32(w),
		biHeight:      int32(-h),
		biPlanes:      1,
		biBitCount:    32,
		biCompression: 0,
		biSizeImage:   uint32(w * h * 4),
	}
	pixels := make([]byte, w*h*4)
	r, _, _ = procGetDIBits.Call(hdc, uintptr(ii.hbmColor), 0, uintptr(h),
		uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&bih)), dibRGBColors)
	if r == 0 {
		return ""
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	hasAlpha := false
	for i := 0; i < w*h; i++ {
		b, g, rr, a := pixels[i*4], pixels[i*4+1], pixels[i*4+2], pixels[i*4+3]
		if a != 0 {
			hasAlpha = true
		}
		img.Pix[i*4] = rr
		img.Pix[i*4+1] = g
		img.Pix[i*4+2] = b
		img.Pix[i*4+3] = a
	}
	if !hasAlpha {
		for i := 0; i < w*h; i++ {
			img.Pix[i*4+3] = 255
			if img.Pix[i*4] == 0 && img.Pix[i*4+1] == 0 && img.Pix[i*4+2] == 0 {
				img.Pix[i*4+3] = 0
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	iconCacheMu.Lock()
	iconCache[path] = data
	iconCacheMu.Unlock()
	return data
}

// procGetIconInfo must be declared separately because it belongs to user32.
var procGetIconInfo = user32DLL.NewProc("GetIconInfo")

func isSystemImage(path string) bool {
	if path == "" {
		return true
	}
	lower := strings.ToLower(path)
	root := strings.ToLower(os.Getenv("SystemRoot"))
	if root == "" {
		root = `c:\windows`
	}
	root = strings.TrimRight(root, `\`) + `\`
	return strings.HasPrefix(lower, root)
}

type appWindow struct {
	Pid  int
	Name string
}

var appWindowList []appWindow

func enumWindowsProc(hwnd uintptr, _ uintptr) uintptr {
	if r, _, _ := procIsWindowVisible.Call(hwnd); r == 0 {
		return 1
	}
	ex, _, _ := procGetWindowLongPtrW.Call(hwnd, uintptr(0xFFFFFFEC))
	if ex&0x00000080 != 0 { // WS_EX_TOOLWINDOW
		return 1
	}
	if ex&0x00000008 != 0 { // WS_EX_NOACTIVATE
		return 1
	}
	var buf [512]uint16
	n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return 1
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return 1
	}
	path := processImagePath(int(pid))
	if isSystemImage(path) {
		return 1
	}
	name := syscall.UTF16ToString(buf[:n])
	if strings.TrimSpace(name) == "" {
		return 1
	}
	appWindowList = append(appWindowList, appWindow{Pid: int(pid), Name: name})
	return 1
}

func enumerateProcessIcons() map[string]any {
	appWindowList = nil
	procEnumWindows.Call(syscall.NewCallback(enumWindowsProc), 0)
	out := make(map[string]any, len(appWindowList))
	for _, w := range appWindowList {
		path := processImagePath(w.Pid)
		ico := iconForPath(path)
		out[strconv.Itoa(w.Pid)] = map[string]any{
			"name": filepath.Base(path),
			"title": w.Name,
			"ico":  ico,
		}
	}
	return out
}

func enumerateAllProcessIcons() map[string]any {
	procs := CommAnd.EnumerateProcesses()
	out := make(map[string]any, len(procs))
	for pid, name := range procs {
		path := processImagePath(pid)
		out[strconv.Itoa(pid)] = map[string]any{
			"name": name,
			"ico":  iconForPath(path),
		}
	}
	return out
}

func iconForPath(path string) string {
	if path == "" {
		return genericIcon()
	}
	if v := extractIconFromPath(path); v != "" {
		return v
	}
	return genericIcon()
}
