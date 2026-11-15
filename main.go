package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// در بیلد با -ldflags "-X main.Version=1.0.0 -X main.UpdateRepo=owner/repo -X main.UpdatePubKey=<hex>" تنظیم می‌شود
var (
	Version      = "dev"
	UpdateRepo   = ""
	UpdatePubKey = ""
)

//go:embed web
var webFS embed.FS

type Config struct {
	Host string `json:"host"` // نام .local → avand.local
	Port int    `json:"port"`
}

type App struct {
	cfg      Config
	db       *DB
	dir      string
	hostname string
	urls     []string
	port     int
	uploads  atomic.Int32
	upd      updateState
}

func (a *App) central() bool {
	a.db.mu.RLock()
	defer a.db.mu.RUnlock()
	return !a.db.CentralOff
}

func loadConfig(dir string) Config {
	c := Config{Host: "avand", Port: 80}
	p := filepath.Join(dir, "config.json")
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &c)
	} else {
		b, _ := json.MarshalIndent(c, "", "  ")
		_ = os.WriteFile(p, b, 0o600)
	}
	c.Host = strings.ToLower(strings.TrimSpace(c.Host))
	if c.Host == "" || strings.ContainsAny(c.Host, " ./\\") {
		c.Host = "avand"
	}
	if c.Port <= 0 || c.Port > 65535 {
		c.Port = 80
	}
	return c
}

func lanIPs() []net.IP {
	var out []net.IP
	ifs, _ := net.Interfaces()
	for _, in := range ifs {
		if in.Flags&net.FlagUp == 0 || in.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := in.Addrs()
		for _, ad := range addrs {
			if ipn, ok := ad.(*net.IPNet); ok {
				if v4 := ipn.IP.To4(); v4 != nil && isPrivate(v4) {
					out = append(out, v4)
				}
			}
		}
	}
	return out
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	page := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			b, _ := fs.ReadFile(sub, name)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; media-src 'self' blob:; style-src 'self' 'unsafe-inline'; frame-ancestors 'self'; base-uri 'self'; form-action 'self'; object-src 'self'; worker-src 'self' blob:")
			_, _ = w.Write(b)
		}
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if a.currentUser(r) != nil {
			http.Redirect(w, r, "/app", http.StatusSeeOther)
			return
		}
		page("login.html")(w, r)
	})
	mux.HandleFunc("GET /app", func(w http.ResponseWriter, r *http.Request) {
		if a.currentUser(r) == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		page("app.html")(w, r)
	})
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		u := a.currentUser(r)
		if u == nil || !u.Admin {
			http.Redirect(w, r, "/app", http.StatusSeeOther)
			return
		}
		page("admin.html")(w, r)
	})
	assets, _ := fs.Sub(sub, "assets")
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assets)))
	mux.Handle("GET /fonts/", http.StripPrefix("/fonts/", http.FileServer(http.Dir(filepath.Join(a.dir, "fonts")))))

	mux.HandleFunc("GET /api/state", a.state)
	mux.HandleFunc("GET /api/vc/browser", a.vcBrowserGet)
	mux.HandleFunc("POST /api/vc/browser", a.vcBrowserPost)
	mux.HandleFunc("GET /api/vc/puzzle", a.vcPuzzleGet)
	mux.HandleFunc("POST /api/vc/puzzle", a.vcPuzzlePost)
	mux.HandleFunc("POST /api/check-code", a.checkCode)
	mux.HandleFunc("POST /register", a.register)
	mux.HandleFunc("POST /login", a.login)
	mux.HandleFunc("POST /setup", a.setup)
	mux.HandleFunc("POST /api/logout", a.logout)
	mux.HandleFunc("GET /api/me", a.me)
	mux.HandleFunc("POST /api/password", a.changePassword)

	a.fileRoutes(mux)
	a.adminRoutes(mux)
	return a.guard(mux)
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		_ = exec.Command("open", url).Start()
	}
}

func main() {
	if len(os.Args) > 1 && handleTool(os.Args[1:]) {
		return
	}
	dir := os.Getenv("AVAND_DATA")
	noBrowser := os.Getenv("AVAND_NO_BROWSER") != ""
	for _, x := range os.Args[1:] {
		if x == "--no-browser" {
			noBrowser = true
		}
	}
	if dir == "" {
		exe, _ := os.Executable()
		dir = filepath.Join(filepath.Dir(exe), "AvandData")
		// نصاب می‌تواند پوشه‌ی داده را جای دیگری (مثلاً درایو D) تعیین کند
		if b, err := os.ReadFile(filepath.Join(filepath.Dir(exe), "avand.path")); err == nil {
			if p := strings.TrimSpace(string(b)); p != "" {
				dir = p
			}
		}
	}
	for _, d := range []string{"", "files", "thumbs", "tmp", "fonts"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o700); err != nil {
			fmt.Println("Cannot create data folder:", err)
			os.Exit(1)
		}
	}
	cleanTmp(filepath.Join(dir, "tmp"))
	db, _ := openDB(dir)
	a := &App{cfg: loadConfig(dir), db: db, dir: dir}
	if p := os.Getenv("AVAND_PORT"); p != "" {
		fmt.Sscan(p, &a.cfg.Port)
	}
	a.hostname, _ = os.Hostname()
	removeOldExe()
	if os.Getenv("AVAND_RESTART") != "1" && os.Getenv("AVAND_PORT") == "" {
		if p := runningInstance(a.cfg.Port); p != 0 { // نمونه‌ی دیگری در حال اجراست: فقط مرورگر را باز کن
			if !noBrowser {
				openBrowser(fmt.Sprintf("http://localhost:%d", p))
			}
			return
		}
	}

	var ln net.Listener
	var err error
	tries := 1
	if os.Getenv("AVAND_RESTART") == "1" { // بعد از به‌روزرسانی: منتظر آزاد شدن پورت قبلی
		tries = 40
	}
	for i := 0; i < tries; i++ {
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", a.cfg.Port))
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	for _, p := range []int{8080, 8081, 8888, 0} {
		if err == nil {
			break
		}
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", p))
	}
	if err != nil {
		fmt.Println("Cannot open any port:", err)
		os.Exit(1)
	}
	a.port = ln.Addr().(*net.TCPAddr).Port
	suffix := ""
	if a.port != 80 {
		suffix = fmt.Sprintf(":%d", a.port)
	}
	a.urls = append(a.urls, "http://"+a.cfg.Host+".local"+suffix)
	for _, ip := range lanIPs() {
		a.urls = append(a.urls, "http://"+ip.String()+suffix)
	}

	srv := &http.Server{Handler: a.routes(), ReadHeaderTimeout: 15 * time.Second, IdleTimeout: 2 * time.Minute}
	go a.startMDNS()
	go a.updateLoop()

	fmt.Printf("\n  AVAND %s — local file center\n  ─────────────────────────────────────────\n", Version)
	fmt.Printf("  This computer : http://localhost%s\n", suffix)
	for _, u := range a.urls {
		fmt.Printf("  Other devices : %s\n", u)
	}
	fmt.Printf("  Data folder   : %s\n\n  Keep this window open. Close it to stop.\n\n", dir)
	if !noBrowser {
		go func() { time.Sleep(600 * time.Millisecond); openBrowser("http://localhost" + suffix) }()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	go func() {
		select {
		case <-stop:
		case <-quitCh:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
	db.Flush()
}

func cleanTmp(dir string) {
	es, _ := os.ReadDir(dir)
	for _, e := range es {
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

var quitCh = make(chan struct{}, 1)

// آیا آوند از قبل روی یکی از پورت‌های معمول اجراست؟
func runningInstance(port int) int {
	cl := &http.Client{Timeout: 700 * time.Millisecond}
	for _, p := range []int{port, 8080, 8081, 8888} {
		r, err := cl.Get(fmt.Sprintf("http://127.0.0.1:%d/api/state", p))
		if err != nil {
			continue
		}
		var j map[string]any
		_ = json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4096)).Decode(&j)
		r.Body.Close()
		if _, ok := j["central"]; ok {
			return p
		}
	}
	return 0
}
