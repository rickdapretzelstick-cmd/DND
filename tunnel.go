package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Tunnel exposes the local server to the internet for free, with no account:
// a Cloudflare quick tunnel (cloudflared), or an SSH reverse tunnel to localhost.run.
type Tunnel struct {
	dir string
	mu  sync.Mutex

	state  string // off, starting, on, error
	url    string
	method string
	err    string
	code   string
	cancel context.CancelFunc
}

type TunnelStatus struct {
	State  string `json:"state"`
	URL    string `json:"url"`
	Method string `json:"method"`
	Error  string `json:"error"`
	Code   string `json:"code"`
}

func NewTunnel(dir string) *Tunnel { return &Tunnel{dir: dir, state: "off"} }

var (
	reCloudflare = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)
	reLocalRun   = regexp.MustCompile(`https://[a-z0-9-]+\.(?:lhr\.life|localhost\.run)`)
)

func (t *Tunnel) Status() TunnelStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return TunnelStatus{t.state, t.url, t.method, t.err, t.code}
}

// Code is the room code remote players must enter ("" while offline).
func (t *Tunnel) Code() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state == "off" || t.state == "error" {
		return ""
	}
	return t.code
}

func (t *Tunnel) set(state, url, method, err string) {
	t.mu.Lock()
	t.state, t.url, t.method, t.err = state, url, method, err
	t.mu.Unlock()
}

func newCode() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func (t *Tunnel) Start(port int) {
	t.mu.Lock()
	if t.state == "starting" || t.state == "on" {
		t.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.state, t.url, t.err, t.method = "starting", "", "", ""
	if t.code == "" {
		t.code = newCode()
	}
	t.mu.Unlock()
	go t.run(ctx, port)
}

func (t *Tunnel) Stop() {
	t.mu.Lock()
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	t.state, t.url, t.err = "off", "", ""
	t.code = "" // a fresh code next time
	t.mu.Unlock()
}

func (t *Tunnel) run(ctx context.Context, port int) {
	var errs []string
	for _, m := range t.methods() {
		if ctx.Err() != nil {
			return
		}
		t.set("starting", "", m.name, "")
		bin, err := m.prepare(ctx)
		if err != nil {
			errs = append(errs, m.name+": "+err.Error())
			continue
		}
		url, err := t.exec(ctx, bin, m, port)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			errs = append(errs, m.name+": "+err.Error())
			continue
		}
		_ = url
		return
	}
	if ctx.Err() == nil {
		t.set("error", "", "", "Couldn't open a tunnel. "+strings.Join(errs, " · "))
	}
}

type tunnelMethod struct {
	name    string
	prepare func(ctx context.Context) (string, error)
	args    func(bin string, port int, dir string) []string
	re      *regexp.Regexp
}

func (t *Tunnel) methods() []tunnelMethod {
	cf := tunnelMethod{
		name:    "cloudflare",
		prepare: func(ctx context.Context) (string, error) { return t.findCloudflared(ctx) },
		args: func(bin string, port int, dir string) []string {
			return []string{"tunnel", "--no-autoupdate", "--url", fmt.Sprintf("http://localhost:%d", port)}
		},
		re: reCloudflare,
	}
	ssh := tunnelMethod{
		name: "localhost.run",
		prepare: func(ctx context.Context) (string, error) {
			p, err := exec.LookPath("ssh")
			if err != nil {
				return "", fmt.Errorf("ssh not installed")
			}
			return p, nil
		},
		args: func(bin string, port int, dir string) []string {
			return []string{"-T", "-o", "StrictHostKeyChecking=accept-new", "-o", "UserKnownHostsFile=" + filepath.Join(dir, "tunnel_known_hosts"),
				"-o", "ServerAliveInterval=30", "-o", "ExitOnForwardFailure=yes", "-o", "BatchMode=yes",
				"-R", fmt.Sprintf("80:localhost:%d", port), "nokey@localhost.run"}
		},
		re: reLocalRun,
	}
	if p := os.Getenv("DND_TUNNEL_CMD"); p != "" { // test hook: any command that prints a trycloudflare URL
		return []tunnelMethod{{name: "custom", prepare: func(context.Context) (string, error) { return p, nil },
			args: func(string, int, string) []string { return nil }, re: reCloudflare}}
	}
	// Prefer an installed cloudflared, then Windows' built-in ssh, then download cloudflared.
	if _, err := t.findCloudflaredLocal(); err == nil {
		return []tunnelMethod{cf, ssh}
	}
	return []tunnelMethod{ssh, cf}
}

func (t *Tunnel) cfName() string {
	if runtime.GOOS == "windows" {
		return "cloudflared.exe"
	}
	return "cloudflared"
}

func (t *Tunnel) findCloudflaredLocal() (string, error) {
	if p, err := exec.LookPath("cloudflared"); err == nil {
		return p, nil
	}
	local := filepath.Join(t.dir, t.cfName())
	if st, err := os.Stat(local); err == nil && st.Size() > 1<<20 {
		return local, nil
	}
	return "", fmt.Errorf("not installed")
}

func (t *Tunnel) findCloudflared(ctx context.Context) (string, error) {
	if p, err := t.findCloudflaredLocal(); err == nil {
		return p, nil
	}
	asset := map[string]string{
		"windows/amd64": "cloudflared-windows-amd64.exe", "windows/arm64": "cloudflared-windows-amd64.exe",
		"linux/amd64": "cloudflared-linux-amd64", "linux/arm64": "cloudflared-linux-arm64",
		"darwin/amd64": "", "darwin/arm64": "",
	}[runtime.GOOS+"/"+runtime.GOARCH]
	if asset == "" {
		return "", fmt.Errorf("no download for %s/%s (install cloudflared or ssh)", runtime.GOOS, runtime.GOARCH)
	}
	local := filepath.Join(t.dir, t.cfName())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://github.com/cloudflare/cloudflared/releases/latest/download/"+asset, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	tmp := local + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return "", fmt.Errorf("download failed: %v", err)
	}
	f.Close()
	if err := os.Rename(tmp, local); err != nil {
		return "", err
	}
	return local, nil
}

// exec runs the tunnel process until it exits or ctx is cancelled, reporting the public URL once seen.
func (t *Tunnel) exec(ctx context.Context, bin string, m tunnelMethod, port int) (string, error) {
	cmd := exec.CommandContext(ctx, bin, m.args(bin, port, t.dir)...)
	hideWindow(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return "", err
	}
	found := make(chan string, 1)
	var tail []string
	go func() {
		sc := bufio.NewScanner(out)
		sent := false
		for sc.Scan() {
			line := sc.Text()
			tail = append(tail, line)
			if len(tail) > 6 {
				tail = tail[1:]
			}
			if u := m.re.FindString(line); u != "" && !sent && !strings.Contains(u, "api.trycloudflare") {
				sent = true
				found <- u
			}
		}
		close(found)
	}()
	select {
	case u, ok := <-found:
		if !ok {
			_ = cmd.Wait()
			return "", fmt.Errorf("exited early: %s", strings.TrimSpace(strings.Join(tail, " ")))
		}
		t.set("on", u, m.name, "")
		werr := cmd.Wait()
		if ctx.Err() == nil {
			t.set("error", "", m.name, "The tunnel dropped ("+fmt.Sprint(werr)+"). Click Go online to reconnect.")
		}
		return u, nil
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		return "", fmt.Errorf("timed out waiting for a public address")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
