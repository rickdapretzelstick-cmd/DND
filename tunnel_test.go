package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rickdapretzelstick-cmd/DND/engine"
)

func TestIsLocalRejectsTunneledRequests(t *testing.T) {
	mk := func(remote, host string, hdr map[string]string) *http.Request {
		r := httptest.NewRequest("GET", "http://"+host+"/", nil)
		r.RemoteAddr = remote
		for k, v := range hdr {
			r.Header.Set(k, v)
		}
		return r
	}
	if !isLocal(mk("127.0.0.1:5000", "localhost:7777", nil)) || !isLocal(mk("[::1]:5000", "127.0.0.1:7777", nil)) {
		t.Fatal("the host's own browser must be local")
	}
	for name, r := range map[string]*http.Request{
		"forwarded":   mk("127.0.0.1:5000", "localhost:7777", map[string]string{"X-Forwarded-For": "1.2.3.4"}),
		"public host": mk("127.0.0.1:5000", "abc.trycloudflare.com", nil),
		"lan":         mk("192.168.1.5:5000", "192.168.1.2:7777", nil),
	} {
		if isLocal(r) {
			t.Fatalf("%s request must not be treated as the host", name)
		}
	}
}

func TestTunnelAndRoomCode(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake.sh")
	_ = os.WriteFile(script, []byte("#!/bin/sh\necho 'INF +--------+'\necho 'INF https://happy-test-1.trycloudflare.com'\nsleep 30\n"), 0o755)
	t.Setenv("DND_TUNNEL_CMD", script)
	srv := NewServer(engine.New(dir), fstest.MapFS{"index.html": {Data: []byte("hi")}})
	srv.tunnel, srv.port = NewTunnel(dir), 7777
	defer srv.tunnel.Stop()

	do := func(path, body string, remote bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://localhost:7777"+path, strings.NewReader(body))
		r.RemoteAddr = "127.0.0.1:4444"
		if remote {
			r.Header.Set("X-Forwarded-For", "9.9.9.9")
			r.Host = "happy-test-1.trycloudflare.com"
		}
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		return w
	}
	if w := do("/api/host/online", `{"action":"start"}`, true); w.Code != 403 {
		t.Fatalf("remote must not control the tunnel, got %d", w.Code)
	}
	do("/api/host/online", `{"action":"start"}`, false)
	deadline := time.Now().Add(5 * time.Second)
	for srv.tunnel.Status().State != "on" && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	st := srv.tunnel.Status()
	if st.State != "on" || st.URL != "https://happy-test-1.trycloudflare.com" || st.Code == "" {
		t.Fatalf("tunnel not up: %+v", st)
	}
	if w := do("/api/join", `{"name":"Sam"}`, true); w.Code != 403 {
		t.Fatalf("remote join without code should be refused, got %d", w.Code)
	}
	if w := do("/api/join", `{"name":"Sam","code":"`+strings.ToLower(st.Code)+`"}`, true); w.Code != 200 {
		t.Fatalf("remote join with code should work, got %d %s", w.Code, w.Body.String())
	}
	if w := do("/api/join", `{"name":"Host"}`, false); w.Code != 200 {
		t.Fatalf("the host joins without a code, got %d", w.Code)
	}
	srv.tunnel.Stop()
	if w := do("/api/join", `{"name":"Late"}`, true); w.Code != 200 {
		t.Fatalf("with the tunnel off, joining needs no code, got %d", w.Code)
	}
}
