package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestSpawn_Smoke: build the castle, start it, spawn a daemon, verify it lives.
func TestSpawn_Smoke(t *testing.T) {
	addr := ":19901"

	t.Log("building castle...")
	buildOut, buildErr := exec.Command("go", "build", "-o", "castle-test", ".").CombinedOutput()
	if buildErr != nil {
		t.Fatalf("go build failed: %s\n%s", buildErr, string(buildOut))
	}
	defer os.Remove("castle-test")

	tmpCastle, err := os.CreateTemp("", "castle-test-*.jsonl")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	tmpCastle.Close()
	defer os.Remove(tmpCastle.Name())

	castleProc := exec.Command("./castle-test", "-serve", addr, tmpCastle.Name())
	castleProc.Stdout = nil
	castleProc.Stderr = nil
	if err := castleProc.Start(); err != nil {
		t.Fatalf("start castle: %v", err)
	}
	defer func() { _ = castleProc.Process.Kill() }()

	// Wait for it to be ready
	for i := 0; i < 30; i++ {
		time.Sleep(200 * time.Millisecond)
		resp, err := http.Get("http://" + addr + "/")
		if err == nil {
			resp.Body.Close()
			break
		}
		if i == 29 {
			t.Fatalf("castle not reachable")
		}
	}

	t.Log("testing /spawn...")
	resp, err := http.Post("http://"+addr+"/spawn", "application/json",
		strings.NewReader(`{"name":"test-spawn","instructions":"confirm alive","brood_secs":10}`))
	if err != nil {
		t.Fatalf("POST /spawn: %v", err)
	}
	defer resp.Body.Close()

	var r struct {
		OK      bool   `json:"ok"`
		PID     int    `json:"pid"`
		Name    string `json:"name"`
		Mailbox string `json:"mailbox"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("/spawn not JSON: %s", string(b))
	}
	if !r.OK {
		t.Fatalf("/spawn error: %s", r.Error)
	}
	if r.PID <= 0 {
		t.Fatalf("invalid PID: %d", r.PID)
	}
	if r.Name != "test-spawn" {
		t.Fatalf("name mismatch: %q", r.Name)
	}

	// Verify process running
	time.Sleep(500 * time.Millisecond)
	psOut, _ := exec.Command("ps", "-p", fmt.Sprintf("%d", r.PID)).CombinedOutput()
	if !strings.Contains(string(psOut), "test-spawn") {
		t.Fatalf("process %d not running: %s", r.PID, string(psOut))
	}

	// Verify mailbox
	mb := os.ExpandEnv(fmt.Sprintf("$HOME/.pilot/mailbox/test-spawn.jsonl"))
	if _, err := os.Stat(mb); err != nil {
		t.Fatalf("mailbox missing: %v", err)
	}
	os.Remove(mb)

	exec.Command("kill", fmt.Sprintf("%d", r.PID)).Run()
	t.Logf("OK /spawn PID=%d", r.PID)
}

// TestPageCompiles: verify the page constant is non-empty.
func TestPageCompiles(t *testing.T) {
	if page == "" {
		t.Fatal("page constant is empty")
	}
	t.Logf("page constant is %d bytes", len(page))
}

// TestWorldJSON: verify key features exist in the page.
func TestWorldJSON(t *testing.T) {
	if !strings.Contains(page, "world.json") {
		t.Fatal("missing /world.json reference")
	}
	if !strings.Contains(page, "minimap") {
		t.Fatal("missing minimap")
	}
	if !strings.Contains(page, "saybar") {
		t.Fatal("missing saybar")
	}
	if !strings.Contains(page, "drawMinimap") {
		t.Fatal("missing drawMinimap function")
	}
}
