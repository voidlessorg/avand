package main

import (
	"embed"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed ui.html
var uiHTML []byte

//go:embed fonts
var fontFS embed.FS

//go:embed avand.exe
var avandExe []byte

var version = "dev"

var token string

func hide(c *exec.Cmd) *exec.Cmd {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return c
}

func openURL(u string) { _ = exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start() }

func psq(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

type req struct {
	Dir, Data           string
	Desk, Menu, Auto, FW bool
	Launch              bool
}

func jerr(w http.ResponseWriter, msg string) {
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
}

var installedExe string

func install(r req) error {
	dir := filepath.Clean(r.Dir)
	if !filepath.IsAbs(dir) || len(dir) < 4 {
		return fmt.Errorf("مسیر نصب نامعتبر است.")
	}
	exe := filepath.Join(dir, "avand.exe")
	_ = hide(exec.Command("taskkill", "/F", "/IM", "avand.exe")).Run()
	time.Sleep(700 * time.Millisecond)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ساخت پوشه‌ی نصب ممکن نشد: %v", err)
	}
	_ = os.Remove(exe + ".old")
	_ = os.Rename(exe, exe+".old")
	if err := os.WriteFile(exe, avandExe, 0o755); err != nil {
		return fmt.Errorf("کپی برنامه ممکن نشد: %v", err)
	}
	if d := strings.TrimSpace(r.Data); d != "" {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return fmt.Errorf("ساخت پوشه‌ی داده ممکن نشد: %v", err)
		}
		_ = os.WriteFile(filepath.Join(dir, "avand.path"), []byte(d), 0o644)
	}
	lnk := func(path string) {
		ps := fmt.Sprintf(`$d=Split-Path %s; New-Item -ItemType Directory -Force -Path $d | Out-Null; $s=(New-Object -ComObject WScript.Shell).CreateShortcut(%s); $s.TargetPath=%s; $s.WorkingDirectory=%s; $s.IconLocation=%s; $s.Description='Avand'; $s.Save()`,
			psq(path), psq(path), psq(exe), psq(dir), psq(exe))
		_ = hide(exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps)).Run()
	}
	if r.Desk {
		if p := os.Getenv("USERPROFILE"); p != "" {
			lnk(filepath.Join(p, "Desktop", "Avand.lnk"))
			// دسکتاپ ممکن است به OneDrive منتقل شده باشد
			if od := os.Getenv("OneDrive"); od != "" {
				if _, err := os.Stat(filepath.Join(od, "Desktop")); err == nil {
					lnk(filepath.Join(od, "Desktop", "Avand.lnk"))
				}
			}
		}
	}
	if r.Menu {
		lnk(filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs\Avand`, "Avand.lnk"))
	}
	const un = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Avand`
	for _, kv := range [][2]string{{"DisplayName", "Avand — آوند"}, {"DisplayVersion", version}, {"Publisher", "Anjam Vazife"}, {"InstallLocation", dir}, {"DisplayIcon", exe},
		{"UninstallString", `"` + exe + `" uninstall /S`}, {"URLInfoAbout", "https://daramet.com/zohalgame"}} {
		_ = hide(exec.Command("reg", "add", un, "/v", kv[0], "/t", "REG_SZ", "/d", kv[1], "/f")).Run()
	}
	run := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	if r.Auto {
		_ = hide(exec.Command("reg", "add", run, "/v", "Avand", "/t", "REG_SZ", "/d", `"`+exe+`" --no-browser`, "/f")).Run()
	} else {
		_ = hide(exec.Command("reg", "delete", run, "/v", "Avand", "/f")).Run()
	}
	if r.FW {
		ps := fmt.Sprintf(`Start-Process netsh -Verb RunAs -WindowStyle Hidden -Wait -ArgumentList 'advfirewall firewall add rule name=Avand dir=in action=allow program="%s" enable=yes profile=private'`, exe)
		_ = hide(exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps)).Run()
	}
	installedExe = exe
	return nil
}

func main() {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	token = hex.EncodeToString(b)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		os.Exit(1)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	done := make(chan struct{})
	auth := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-K") != token {
				http.Error(w, "forbidden", 403)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			h(w, r)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("k") != token {
			http.Error(w, "forbidden", 403)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'; img-src data:; frame-ancestors 'none'")
		_, _ = w.Write(uiHTML)
	})
	mux.Handle("GET /fonts/", http.FileServerFS(fontFS))
	mux.HandleFunc("GET /api/defaults", auth(func(w http.ResponseWriter, r *http.Request) {
		dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Avand")
		data := filepath.Join(os.Getenv("USERPROFILE"), "AvandData")
		if _, err := os.Stat(`D:\`); err == nil {
			data = `D:\AvandData`
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"dir": dir, "data": data})
	}))
	mux.HandleFunc("POST /api/install", auth(func(w http.ResponseWriter, r *http.Request) {
		var q req
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&q) != nil {
			jerr(w, "درخواست نامعتبر")
			return
		}
		if err := install(q); err != nil {
			jerr(w, err.Error())
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	mux.HandleFunc("POST /api/finish", auth(func(w http.ResponseWriter, r *http.Request) {
		var q req
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&q)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		if q.Launch && installedExe != "" {
			c := exec.Command(installedExe)
			c.Dir = filepath.Dir(installedExe)
			_ = c.Start()
		}
		go func() { time.Sleep(400 * time.Millisecond); close(done) }()
	}))
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)
	openURL(fmt.Sprintf("http://127.0.0.1:%d/?k=%s", port, token))
	select {
	case <-done:
	case <-time.After(30 * time.Minute):
	}
}
