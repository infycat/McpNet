package main

import "testing"

func TestMatchingProcessesUsesExactExecutableNameAndStablePIDOrder(t *testing.T) {
	got := matchingProcesses("MiniGameApp.exe", map[int]string{
		36116: `C:\Games\MiniGameApp.exe`,
		88:    `C:\Games\other.exe`,
		7:     `D:\Alt\minigameapp.EXE`,
	})
	if len(got) != 2 {
		t.Fatalf("got %d matching processes, want 2: %#v", len(got), got)
	}
	if got[0]["pid"] != 7 || got[1]["pid"] != 36116 {
		t.Fatalf("PIDs are not sorted/stable: %#v", got)
	}
	if got[0]["name"] != "minigameapp.EXE" {
		t.Fatalf("name = %#v, want executable basename", got[0]["name"])
	}
}

func TestMatchingProcessesRejectsBlankName(t *testing.T) {
	if got := matchingProcesses("  ", map[int]string{1: "MiniGameApp.exe"}); got != nil {
		t.Fatalf("blank filter returned processes: %#v", got)
	}
}

func TestMCPProcessToolsAreAdvertised(t *testing.T) {
	want := map[string]bool{"mcpnet_list_processes": false, "mcpnet_process_control": false}
	for _, tool := range mcpTools() {
		if _, exists := want[tool.Name]; exists {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("MCP tool %q is not advertised", name)
		}
	}
}

func TestProcessDriverLoadFailsClearlyWithoutCore(t *testing.T) {
	if _, err := (&App{}).startProcessDriver(); err == nil {
		t.Fatal("expected an error when the McpNet core is not initialized")
	}
}
