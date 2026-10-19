//go:build windows

package main

import (
	"log"
	"syscall"
	"unsafe"
)

var (
	modUser32                 = syscall.NewLazyDLL("user32.dll")
	procFindWindowW           = modUser32.NewProc("FindWindowW")
	procGetWindowRect         = modUser32.NewProc("GetWindowRect")
	procSetWindowPos          = modUser32.NewProc("SetWindowPos")
	procGetDpiForWindow       = modUser32.NewProc("GetDpiForWindow")
	procSystemParametersInfoW = modUser32.NewProc("SystemParametersInfoW")
)

type winRect struct {
	Left, Top, Right, Bottom int32
}

const (
	spiGetWorkArea = 0x0030
	swpNoZOrder    = 0x0004
	swpShowWindow  = 0x0040
)

// fitWindowToWorkArea places the main window fully inside the primary monitor's
// work area, shrinking it when necessary.
//
// At 125% DPI scaling a 1176x700 logical window becomes 1470x875 physical. The
// window then sits at y=133, so its bottom edge is at 1008 on a 960px screen and
// the bottom status bar is drawn off-screen.
func fitWindowToWorkArea() {
	title, err := syscall.UTF16PtrFromString("McpNet")
	if err != nil {
		return
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return
	}
	var wa winRect
	if ret, _, _ := procSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&wa)), 0); ret == 0 {
		return
	}
	aw := int(wa.Right - wa.Left)
	ah := int(wa.Bottom - wa.Top)
	if aw <= 0 || ah <= 0 {
		return
	}
	var cur winRect
	if ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&cur))); ret == 0 {
		return
	}
	cw := int(cur.Right - cur.Left)
	ch := int(cur.Bottom - cur.Top)

	// Keep a small margin so window borders and shadows stay visible.
	targetW := cw
	targetH := ch
	if targetW > aw-16 {
		targetW = aw - 16
	}
	if targetH > ah-16 {
		targetH = ah - 16
	}

	fits := cur.Left >= wa.Left && cur.Top >= wa.Top &&
		int(cur.Right) <= int(wa.Right) && int(cur.Bottom) <= int(wa.Bottom)
	if fits {
		return
	}

	x := int(wa.Left) + (aw-targetW)/2
	y := int(wa.Top) + (ah-targetH)/2
	if x < int(wa.Left) {
		x = int(wa.Left)
	}
	if y < int(wa.Top) {
		y = int(wa.Top)
	}
	procSetWindowPos.Call(hwnd, 0,
		uintptr(x), uintptr(y), uintptr(targetW), uintptr(targetH),
		uintptr(swpNoZOrder|swpShowWindow))

	dpi, _, _ := procGetDpiForWindow.Call(hwnd)
	log.Printf("Window did not fit the work area (%d,%d %dx%d vs %dx%d at dpi %d); moved to %d,%d size %dx%d",
		cur.Left, cur.Top, cw, ch, aw, ah, dpi, x, y, targetW, targetH)
}
