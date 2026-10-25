package main

import (
	"changeme/CommAnd"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qtgolang/McpNet/src/JsCall"
	"github.com/qtgolang/McpNet/src/protobuf/JSON"
)

func processEvent(command string, args *JSON.SyJson) any {
	switch command {
	case "加载驱动":
		loaded, _ := app.startProcessDriver()
		return loaded
	case "枚举进程":
		p := CommAnd.EnumerateProcesses()
		return p
	case "枚举进程图标":
		return enumerateProcessIcons()
	case "枚举全部进程图标":
		return enumerateAllProcessIcons()
	case "MCP進程篩選狀態":
		return app.processFilterSnapshot()
	case "MCP進程驅動狀態":
		return app.processDriverStatus()
	case "进程驱动添加PID":
		gx := args.GetData("isSelected") == "true"
		return app.setProcessPIDFilter(getInt(args.GetData("PID")), gx, true)
	case "进程驱动添加进程名":
		gx := args.GetData("isSet") == "true"
		return app.setProcessNameFilter(args.GetData("Name"), gx)
	default:
		return ReplaceRulesEvent(command, args)
	}
}

func (a *App) setProcessPIDFilter(pid int, selected, replace bool) bool {
	if a == nil || a.App == nil || pid <= 0 {
		return false
	}
	a.processFilterMu.Lock()
	if a.processPIDFilters == nil {
		a.processPIDFilters = make(map[int]bool)
	}
	if a.processNameFilters == nil {
		a.processNameFilters = make(map[string]bool)
	}
	if replace {
		a.App.ProcessCancelAll()
		clear(a.processPIDFilters)
		clear(a.processNameFilters)
		a.processAllFiltered = false
	}
	if selected {
		a.App.ProcessAddPid(pid)
		a.processPIDFilters[pid] = true
	} else {
		a.App.ProcessDelPid(pid)
		delete(a.processPIDFilters, pid)
	}
	a.processFilterMu.Unlock()
	a.notifyMCPUIStateChanged("process")
	return true
}

func (a *App) setProcessPIDFilterMCP(pid int, selected bool) bool {
	return a.setProcessPIDFilter(pid, selected, false)
}

func (a *App) setProcessNameFilter(name string, selected bool) bool {
	if a == nil || a.App == nil {
		return false
	}
	name = strings.TrimSpace(name)
	a.processFilterMu.Lock()
	if a.processPIDFilters == nil {
		a.processPIDFilters = make(map[int]bool)
	}
	if a.processNameFilters == nil {
		a.processNameFilters = make(map[string]bool)
	}
	if name == "{OpenALL}" {
		a.App.ProcessALLName(selected, false)
		a.processAllFiltered = selected
	} else {
		if selected {
			a.App.ProcessAddName(JsCall.ToGBK(name))
			a.processNameFilters[strings.ToLower(name)] = true
		} else {
			a.App.ProcessDelName(JsCall.ToGBK(name))
			delete(a.processNameFilters, strings.ToLower(name))
		}
	}
	a.processFilterMu.Unlock()
	a.notifyMCPUIStateChanged("process")
	return true
}

func (a *App) clearProcessFilters() bool {
	if a == nil || a.App == nil {
		return false
	}
	a.processFilterMu.Lock()
	a.App.ProcessCancelAll()
	clear(a.processPIDFilters)
	clear(a.processNameFilters)
	a.processAllFiltered = false
	a.processFilterMu.Unlock()
	a.notifyMCPUIStateChanged("process")
	return true
}

func (a *App) processFilterSnapshot() map[string]any {
	a.processFilterMu.RLock()
	pids := make([]int, 0, len(a.processPIDFilters))
	for pid := range a.processPIDFilters {
		pids = append(pids, pid)
	}
	names := make([]string, 0, len(a.processNameFilters))
	for name := range a.processNameFilters {
		names = append(names, name)
	}
	all := a.processAllFiltered
	a.processFilterMu.RUnlock()
	sort.Ints(pids)
	sort.Strings(names)
	return map[string]any{"pids": pids, "names": names, "all": all}
}

func (a *App) startProcessDriver() (bool, error) {
	a.processDriverMu.Lock()
	defer a.processDriverMu.Unlock()
	if a.App == nil {
		return false, fmt.Errorf("McpNet core is not initialized")
	}
	if a.processDriverLoaded {
		return true, nil
	}
	if !a.App.StartProcess() {
		return false, fmt.Errorf("process interception driver failed to load; run McpNet with required Windows privileges and inspect the startup log")
	}
	a.processDriverLoaded = true
	return true, nil
}

func (a *App) processDriverStatus() bool {
	a.processDriverMu.Lock()
	defer a.processDriverMu.Unlock()
	return a.processDriverLoaded
}

func matchingProcesses(name string, processes map[int]string) []map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	pids := make([]int, 0, len(processes))
	for pid, executable := range processes {
		if strings.EqualFold(filepath.Base(executable), name) {
			pids = append(pids, pid)
		}
	}
	sort.Ints(pids)
	out := make([]map[string]any, 0, len(pids))
	for _, pid := range pids {
		out = append(out, map[string]any{"pid": pid, "name": filepath.Base(processes[pid])})
	}
	return out
}
