package main

import (
	"bytes"
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

// buildCastleAndPilot builds castle-test and, beside it, the pilot daemon:
// /spawn resolves the daemon next to the castle binary, then PATH (serve.go),
// and a test or CI runner has neither. Same layout build.sh produces.
func buildCastleAndPilot(t *testing.T) {
	t.Helper()
	t.Log("building castle + pilot...")
	if out, err := exec.Command("go", "build", "-o", "castle-test", ".").CombinedOutput(); err != nil {
		t.Fatalf("go build castle failed: %s\n%s", err, string(out))
	}
	t.Cleanup(func() { os.Remove("castle-test") })
	if out, err := exec.Command("go", "build", "-o", "pilot", "../..").CombinedOutput(); err != nil {
		t.Fatalf("go build pilot failed: %s\n%s", err, string(out))
	}
	t.Cleanup(func() { os.Remove("pilot") })
}

// startCastle serves a castle on addr over a temp world file and waits for it.
func startCastle(t *testing.T, addr string) {
	t.Helper()
	world, err := os.CreateTemp("", "castle-test-*.jsonl")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	world.Close()
	t.Cleanup(func() { os.Remove(world.Name()) })
	proc := exec.Command("./castle-test", "-serve", addr, world.Name())
	if err := proc.Start(); err != nil {
		t.Fatalf("start castle: %v", err)
	}
	t.Cleanup(func() { _ = proc.Process.Kill() })
	for i := 0; ; i++ {
		time.Sleep(200 * time.Millisecond)
		if resp, err := http.Get("http://" + addr + "/"); err == nil {
			resp.Body.Close()
			return
		}
		if i == 29 {
			t.Fatalf("castle not reachable")
		}
	}
}

type spawnReply struct {
	OK      bool   `json:"ok"`
	PID     int    `json:"pid"`
	Name    string `json:"name"`
	Mailbox string `json:"mailbox"`
	Error   string `json:"error"`
}

// spawn POSTs /spawn and decodes the reply. The caller owns the PID.
func spawn(t *testing.T, addr, name string) spawnReply {
	t.Helper()
	resp, err := http.Post("http://"+addr+"/spawn", "application/json",
		strings.NewReader(fmt.Sprintf(`{"name":%q,"instructions":"confirm alive","brood_secs":10}`, name)))
	if err != nil {
		t.Fatalf("POST /spawn: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r spawnReply
	if err := json.Unmarshal(body, &r); err != nil {
		t.Fatalf("/spawn not JSON (status %d): %s", resp.StatusCode, string(body))
	}
	return r
}

// daemonCanStart runs the built daemon directly, with the args the castle
// uses, for a bounded window. pilot/main.go exits 1 naming the missing
// prerequisite — a brain key (DEEPSEEK_API_KEY or ~/.pilot.env) or the
// http-mcp tool-server (HTTP_MCP_BIN / PATH / ../http-mcp) — within ~2s
// (its startup substrate checks alone can take ~2s); a bare CI runner has
// neither. Returns the daemon's own reason when it cannot start here.
func daemonCanStart(t *testing.T) (bool, string) {
	t.Helper()
	world, err := os.CreateTemp("", "castle-preflight-*.jsonl")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	world.Close()
	defer os.Remove(world.Name())
	var stderr bytes.Buffer
	cmd := exec.Command("./pilot", "--daemon", "-name", "preflight", "-castle", world.Name(),
		"-brood", "1")
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return false, err.Error()
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
		reason, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n")
		if reason == "" {
			reason = "exited without a message"
		}
		return false, reason
	case <-time.After(6 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		return true, ""
	}
}

// TestSpawn_Smoke: the /spawn contract — JSON reply, ok, a PID, the name
// echoed, the bootstrap mailbox written before the spawn. Holds on any host:
// whether the daemon then LIVES is the host's business (TestSpawn_DaemonLives).
func TestSpawn_Smoke(t *testing.T) {
	buildCastleAndPilot(t)
	startCastle(t, ":19901")

	t.Log("testing /spawn...")
	r := spawn(t, ":19901", "test-spawn")
	if r.PID > 0 {
		defer exec.Command("kill", fmt.Sprintf("%d", r.PID)).Run()
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
	mb := os.ExpandEnv("$HOME/.pilot/mailbox/test-spawn.jsonl")
	if _, err := os.Stat(mb); err != nil {
		t.Fatalf("mailbox missing: %v", err)
	}
	os.Remove(mb)
	t.Logf("OK /spawn PID=%d mailbox=%s", r.PID, r.Mailbox)
}

// TestSpawn_DaemonLives: a spawned daemon is still running well after its
// startup probes (which take up to ~2s to fail). Skips, naming the daemon's
// own reason, on a host where the daemon cannot start at all — a 500ms check
// used to pass here by timing luck, before the daemon had finished dying.
func TestSpawn_DaemonLives(t *testing.T) {
	buildCastleAndPilot(t)
	if ok, why := daemonCanStart(t); !ok {
		t.Skipf("pilot daemon cannot start on this host: %s — liveness not verifiable here (the /spawn contract is TestSpawn_Smoke)", why)
	}
	startCastle(t, ":19903")

	r := spawn(t, ":19903", "test-spawn-live")
	if r.PID > 0 {
		defer exec.Command("kill", fmt.Sprintf("%d", r.PID)).Run()
	}
	defer os.Remove(os.ExpandEnv("$HOME/.pilot/mailbox/test-spawn-live.jsonl"))
	if !r.OK || r.PID <= 0 {
		t.Fatalf("/spawn failed: ok=%v pid=%d err=%s", r.OK, r.PID, r.Error)
	}
	time.Sleep(4 * time.Second) // past every startup probe
	// -o args=: the full command on Linux too (bare ps -p prints only the comm)
	psOut, _ := exec.Command("ps", "-p", fmt.Sprintf("%d", r.PID), "-o", "args=").CombinedOutput()
	if !strings.Contains(string(psOut), "test-spawn-live") {
		t.Fatalf("process %d not running after 4s: %q", r.PID, string(psOut))
	}
	t.Logf("OK daemon PID=%d alive", r.PID)
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
