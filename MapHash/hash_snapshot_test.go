package MapHash

import (
	"testing"

	"github.com/qtgolang/McpNet/src/http"
)

func TestSnapshotCapturesAreSortedAndDetached(t *testing.T) {
	m := NewHashMap()
	a := &Request{Display: true, Method: "GET", URL: "http://example.test/a", Body: []byte("request")}
	a.Header = http.Header{"X-Test": []string{"one"}}
	b := &Request{Display: true, Way: "Websocket", SocketData: []*UpdateSocketData{{
		Info: &UpdateSocketList{Ico: "上行", WsType: "JSON", Length: 2}, Body: []byte("{}"),
	}}}
	m.SetRequest(9, a)
	m.SetRequest(3, b)

	snapshots := m.SnapshotCaptures()
	if len(snapshots) != 2 || snapshots[0].ID != 3 || snapshots[1].ID != 9 {
		t.Fatalf("unexpected sorted snapshots: %#v", snapshots)
	}
	if snapshots[0].Protocol != "Websocket" || len(snapshots[0].Messages) != 1 || snapshots[0].Messages[0].Direction != "client_to_server" {
		t.Fatalf("unexpected message snapshot: %#v", snapshots[0])
	}
	snapshots[1].RequestBody[0] = 'X'
	snapshots[1].RequestHead["X-Test"][0] = "changed"
	if string(a.Body) != "request" || a.Header["X-Test"][0] != "one" {
		t.Fatal("snapshot mutation changed the live capture")
	}
}

func TestSnapshotCaptureSkipsHiddenAndMissing(t *testing.T) {
	m := NewHashMap()
	m.SetRequest(7, &Request{Display: false})
	if _, ok := m.SnapshotCapture(7); ok {
		t.Fatal("hidden capture must not be exposed")
	}
	if _, ok := m.SnapshotCapture(8); ok {
		t.Fatal("missing capture must not be exposed")
	}
}
