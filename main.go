// Command dnd-runner hosts a multiplayer tabletop campaign on your computer.
// Players join from any browser on the network; the Dungeon Master is either
// Claude (with an API key) or a built-in offline DM.
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/rickdapretzelstick-cmd/DND/engine"
)

//go:embed web
var webFS embed.FS

var version = "dev"

func init() {
	// Windows takes MIME types from the registry, which can break ES modules.
	_ = mime.AddExtensionType(".js", "text/javascript; charset=utf-8")
	_ = mime.AddExtensionType(".css", "text/css; charset=utf-8")
	_ = mime.AddExtensionType(".html", "text/html; charset=utf-8")
}

func main() {
	port := flag.Int("port", 0, "port to listen on (default 7777)")
	dataDir := flag.String("data", "", "folder for saves and settings (default: next to the program)")
	noBrowser := flag.Bool("no-browser", false, "don't open a browser window on start")
	flag.Parse()

	dir := *dataDir
	if dir == "" {
		exe, err := os.Executable()
		if err == nil {
			dir = filepath.Dir(exe)
		} else {
			dir = "."
		}
	}
	if !writable(dir) {
		// e.g. the exe lives in Program Files: keep saves in the user's app data instead.
		if cfg, err := os.UserConfigDir(); err == nil {
			dir = filepath.Join(cfg, "DnD-Campaign-Runner")
		}
		if !writable(dir) {
			fmt.Println("Can't write saves to", dir)
			waitForEnter()
			os.Exit(1)
		}
	}

	eng := engine.New(dir)
	p := *port
	if p == 0 {
		p = eng.Cfg.Port
	}
	if p == 0 {
		p = 7777
	}

	static, _ := fs.Sub(webFS, "web")
	srv := NewServer(eng, static)
	srv.tunnel, srv.port = NewTunnel(dir), p

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
	if err != nil {
		fmt.Printf("\nCouldn't start on port %d (%v).\nIs D&D Campaign Runner already running? Close it, or start with -port 7778.\n", p, err)
		waitForEnter()
		os.Exit(1)
	}

	local := fmt.Sprintf("http://localhost:%d", p)
	fmt.Println("==============================================")
	fmt.Println("   D&D Campaign Runner", version)
	fmt.Println("==============================================")
	fmt.Println()
	fmt.Println(" You (the host) play at:  ", local)
	if ips := lanIPs(); len(ips) > 0 {
		fmt.Println(" Friends on your network join at:")
		for _, ip := range ips {
			fmt.Printf("     http://%s:%d\n", ip, p)
		}
	}
	fmt.Println()
	fmt.Println(" Saves are stored in:", filepath.Join(dir, "saves"))
	fmt.Println(" Keep this window open while you play. Close it to stop the server.")
	fmt.Println()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		fmt.Println("Saving and shutting down...")
		_ = eng.Save()
		os.Exit(0)
	}()

	if !*noBrowser {
		go func() {
			time.Sleep(600 * time.Millisecond)
			openBrowser(local)
		}()
	}
	if err := http.Serve(ln, srv); err != nil {
		log.Println("server stopped:", err)
		_ = eng.Save()
		waitForEnter()
	}
}

func writable(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, ".write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

func lanIPs() []string {
	var out []string
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				if ip4 := ipn.IP.To4(); ip4 != nil && !ip4.IsLinkLocalUnicast() {
					out = append(out, ip4.String())
				}
			}
		}
	}
	return out
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func waitForEnter() {
	fmt.Println("Press Enter to exit.")
	var b [1]byte
	_, _ = os.Stdin.Read(b[:])
}
