package main

// به‌روزرسانی خودکار از GitHub Releases.
// امنیت: هر نسخه با کلید خصوصی ed25519 مدیر پروژه امضا می‌شود و برنامه فقط با کلید عمومیِ جاسازی‌شده می‌پذیرد؛
// پس حتی اگر شبکه یا حساب GitHub دست کس دیگری بیفتد، بدون کلید خصوصی نمی‌شود کدی روی کامپیوتر کاربران اجرا کرد.

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type updateState struct {
	mu      sync.Mutex
	Checked int64  `json:"checked"`
	Latest  string `json:"latest"`
	Status  string `json:"status"` // idle | uptodate | downloading | restarting | error | not-configured | off
	Err     string `json:"error"`
}

func (u *updateState) set(status, latest, err string) {
	u.mu.Lock()
	u.Status, u.Err, u.Checked = status, err, now()
	if latest != "" {
		u.Latest = latest
	}
	u.mu.Unlock()
}
func (u *updateState) view() map[string]any {
	u.mu.Lock()
	defer u.mu.Unlock()
	return map[string]any{"checked": u.Checked, "latest": u.Latest, "status": u.Status, "error": u.Err}
}

func semver(s string) [3]int {
	var out [3]int
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	for i, p := range strings.SplitN(s, ".", 3) {
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		out[i] = n
	}
	return out
}

func newer(a, b string) bool {
	x, y := semver(a), semver(b)
	for i := 0; i < 3; i++ {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

func releaseMessage(tag, sum string) []byte { return []byte("avand-release\n" + tag + "\n" + sum) }

func (a *App) updateLoop() {
	time.Sleep(90 * time.Second)
	for {
		a.db.mu.RLock()
		auto := !a.db.CentralOff && !a.db.NoAutoUpd
		a.db.mu.RUnlock()
		if auto {
			a.checkUpdate(true)
		}
		time.Sleep(6 * time.Hour)
	}
}

func (a *App) checkUpdate(apply bool) {
	if UpdateRepo == "" || UpdatePubKey == "" {
		a.upd.set("not-configured", "", "")
		return
	}
	if Version == "dev" {
		a.upd.set("uptodate", "", "")
		return
	}
	if err := a.doUpdate(apply); err != nil {
		a.upd.set("error", "", err.Error())
	}
}

func (a *App) doUpdate(apply bool) error {
	pub, err := hex.DecodeString(UpdatePubKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("کلید عمومی به‌روزرسانی نامعتبر است")
	}
	base := os.Getenv("AVAND_UPDATE_API")
	testMode := base != ""
	if base == "" {
		base = "https://api.github.com"
	}
	cl := &http.Client{Timeout: 25 * time.Second}
	req, _ := http.NewRequest("GET", base+"/repos/"+UpdateRepo+"/releases/latest", nil)
	req.Header.Set("User-Agent", "Avand-Updater/"+Version)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := cl.Do(req)
	if err != nil {
		return fmt.Errorf("اتصال به GitHub برقرار نشد: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("پاسخ GitHub: %d", resp.StatusCode)
	}
	var rel struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel) != nil {
		return errors.New("پاسخ GitHub قابل خواندن نیست")
	}
	if !newer(rel.Tag, Version) {
		a.upd.set("uptodate", rel.Tag, "")
		return nil
	}
	name := "avand-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var binURL, sigURL string
	for _, as := range rel.Assets {
		if as.Name == name {
			binURL = as.URL
		}
		if as.Name == name+".sig" {
			sigURL = as.URL
		}
	}
	if binURL == "" || sigURL == "" {
		return errors.New("فایل نسخه‌ی جدید یا امضایش در Release نیست")
	}
	for _, u := range []string{binURL, sigURL} {
		if !strings.HasPrefix(u, "https://") && !testMode {
			return errors.New("آدرس دانلود ناامن است")
		}
	}
	a.upd.set("downloading", rel.Tag, "")
	sigB, err := fetchLimited(cl, sigURL, 4096)
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigB)))
	if err != nil {
		return errors.New("امضا نامعتبر است")
	}
	tmp := filepath.Join(a.dir, "tmp", "update-"+randHex(4))
	sum, err := downloadTo(&http.Client{Timeout: 30 * time.Minute}, binURL, tmp, 400<<20)
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if !ed25519.Verify(pub, releaseMessage(rel.Tag, sum), sig) {
		os.Remove(tmp)
		return errors.New("امضای نسخه‌ی جدید معتبر نیست؛ نصب نشد")
	}
	if !apply {
		os.Remove(tmp)
		a.upd.set("uptodate", rel.Tag, "")
		return nil
	}
	for i := 0; i < 120 && a.uploads.Load() > 0; i++ { // وسط آپلود ری‌استارت نکن
		time.Sleep(5 * time.Second)
	}
	a.upd.set("restarting", rel.Tag, "")
	return a.replaceSelf(tmp)
}

func fetchLimited(cl *http.Client, url string, max int64) ([]byte, error) {
	resp, err := cl.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("دانلود ناموفق: %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

func downloadTo(cl *http.Client, url, dst string, max int64) (string, error) {
	resp, err := cl.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("دانلود ناموفق: %d", resp.StatusCode)
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, max+1))
	if err != nil {
		return "", err
	}
	if n > max {
		return "", errors.New("فایل بیش از حد بزرگ است")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (a *App) replaceSelf(newFile string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.EvalSymlinks(exe)
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil { // ویندوز اجازه‌ی rename فایل در حال اجرا را می‌دهد
		return err
	}
	src, err := os.Open(newFile)
	if err != nil {
		_ = os.Rename(old, exe)
		return err
	}
	dst, err := os.OpenFile(exe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		src.Close()
		_ = os.Rename(old, exe)
		return err
	}
	_, err = io.Copy(dst, src)
	src.Close()
	dst.Close()
	os.Remove(newFile)
	if err != nil {
		_ = os.Remove(exe)
		_ = os.Rename(old, exe)
		return err
	}
	a.db.Flush()
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), "AVAND_RESTART=1", "AVAND_NO_BROWSER=1")
	if err := cmd.Start(); err != nil {
		_ = os.Remove(exe)
		_ = os.Rename(old, exe)
		return err
	}
	os.Exit(0)
	return nil
}

func removeOldExe() {
	if exe, err := os.Executable(); err == nil {
		_ = os.Remove(exe + ".old")
	}
}

// ابزارهای مدیر پروژه: avand keygen | avand sign <فایل> <تگ> <کلید-خصوصی-hex> | avand version
func handleTool(args []string) bool {
	switch args[0] {
	case "version":
		fmt.Println(Version)
		return true
	case "uninstall":
		uninstallSelf(len(args) > 1 && strings.EqualFold(args[1], "/S"))
		return true
	case "keygen":
		pub, priv, _ := ed25519.GenerateKey(nil)
		fmt.Println("PUBLIC  (in build -ldflags):", hex.EncodeToString(pub))
		fmt.Println("PRIVATE (keep secret!)     :", hex.EncodeToString(priv))
		return true
	case "sign":
		if len(args) != 4 {
			fmt.Println("usage: avand sign <file> <tag> <private-hex>")
			os.Exit(2)
		}
		b, err := os.ReadFile(args[1])
		priv, err2 := hex.DecodeString(args[3])
		if err != nil || err2 != nil || len(priv) != ed25519.PrivateKeySize {
			fmt.Println("bad input")
			os.Exit(2)
		}
		h := sha256.Sum256(b)
		sig := ed25519.Sign(priv, releaseMessage(args[2], hex.EncodeToString(h[:])))
		_ = os.WriteFile(args[1]+".sig", []byte(base64.StdEncoding.EncodeToString(sig)), 0o644)
		fmt.Println("wrote", args[1]+".sig")
		return true
	}
	return false
}

var _ = strconv.Itoa
