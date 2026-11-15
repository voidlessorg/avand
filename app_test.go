package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type client struct {
	t   *testing.T
	srv *httptest.Server
	jar map[string]string
}

func (c *client) do(method, path string, body any, hdr ...string) (int, map[string]any, *http.Response) {
	var rd io.Reader
	ct := "application/json"
	switch b := body.(type) {
	case nil:
	case []byte:
		rd, ct = bytes.NewReader(b), "application/octet-stream"
	default:
		j, _ := json.Marshal(b)
		rd = bytes.NewReader(j)
	}
	req, _ := http.NewRequest(method, c.srv.URL+path, rd)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Origin", c.srv.URL)
	for i := 0; i+1 < len(hdr); i += 2 {
		if hdr[i+1] == "" {
			req.Header.Del(hdr[i])
		} else {
			req.Header.Set(hdr[i], hdr[i+1])
		}
	}
	for k, v := range c.jar {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	cl := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := cl.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	for _, ck := range resp.Cookies() {
		if ck.MaxAge < 0 {
			delete(c.jar, ck.Name)
		} else {
			c.jar[ck.Name] = ck.Value
		}
	}
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if m == nil {
		m = map[string]any{"_raw": string(raw)}
	}
	return resp.StatusCode, m, resp
}

func newApp(t *testing.T) (*App, *httptest.Server) {
	t.Helper()
	dir := t.TempDir()
	for _, d := range []string{"files", "thumbs", "tmp", "fonts"} {
		_ = mkdir(dir + "/" + d)
	}
	db, _ := openDB(dir)
	a := &App{cfg: Config{Host: "avand", Port: 80}, db: db, dir: dir, hostname: "pc1"}
	srv := httptest.NewServer(a.routes())
	t.Cleanup(srv.Close)
	return a, srv
}

func eq(t *testing.T, got, want any, msg string) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("%s: got %v want %v", msg, got, want)
	}
}

func human(target float64) map[string]any {
	var tr [][3]float64
	tm := 0.0
	for i := 0; i <= 28; i++ {
		k := float64(i) / 28
		e := 1 - (1-k)*(1-k)*(1-k)
		tr = append(tr, [3]float64{tm, float64(int(target*e*10)) / 10, float64(i%3) + 0.5*float64(i%2)})
		tm += float64(14 + (i*7)%23)
	}
	return map[string]any{"x": target, "track": tr, "dur": tm}
}

func TestFullFlow(t *testing.T) {
	a, srv := newApp(t)
	admin := &client{t, srv, map[string]string{}}

	// راه‌اندازی اولیه: فقط یک بار
	_, st, _ := admin.do("GET", "/api/state", nil)
	eq(t, st["setup"], true, "setup needed")
	code, _, _ := admin.do("POST", "/setup", map[string]string{"username": "boss", "password": "password1"})
	eq(t, code, 200, "setup")
	code, _, _ = (&client{t, srv, map[string]string{}}).do("POST", "/setup", map[string]string{"username": "second", "password": "password1"})
	eq(t, code, 400+9, "second setup rejected") // 409
	code, m, _ := admin.do("GET", "/api/me", nil)
	eq(t, code, 200, "me")
	eq(t, m["admin"], true, "admin flag")

	// کد دسترسی با سهمیه ۱۰۰۰ بایت
	_, m, _ = admin.do("POST", "/api/admin/codes", map[string]any{"count": 1, "quota": 1000, "note": "تست"})
	acode := m["codes"].([]any)[0].(string)

	// کاربر عادی: بدون بررسی مرورگر/کپچا نمی‌تواند ثبت‌نام کند (شبکه‌ی مرکزی روشن)
	u := &client{t, srv, map[string]string{}}
	_, m, _ = u.do("POST", "/api/check-code", map[string]string{"code": acode})
	eq(t, m["code"], "bc", "needs browser check")

	// بررسی مرورگر
	_, ch, _ := u.do("GET", "/api/vc/browser", nil)
	seed, bits := ch["seed"].(string), int(ch["bits"].(float64))
	var nonce int
	for ; ; nonce++ {
		if leadingZeroBits(sha256.Sum256([]byte(fmt.Sprintf("%s:%d", seed, nonce)))) >= bits {
			break
		}
	}
	code, _, _ = u.do("POST", "/api/vc/browser", map[string]any{"id": ch["id"], "nonce": fmt.Sprint(nonce), "sig": map[string]any{"wd": true}})
	eq(t, code, 403, "webdriver rejected")
	_, ch, _ = u.do("GET", "/api/vc/browser", nil)
	seed = ch["seed"].(string)
	for nonce = 0; leadingZeroBits(sha256.Sum256([]byte(fmt.Sprintf("%s:%d", seed, nonce)))) < bits; nonce++ {
	}
	code, _, _ = u.do("POST", "/api/vc/browser", map[string]any{"id": ch["id"], "nonce": fmt.Sprint(nonce), "sig": map[string]any{"wd": false}})
	eq(t, code, 200, "browser check ok")

	// پازل: جواب سمت سرور؛ حرکت خطی ربات رد می‌شود
	puzzle := func() (string, int) {
		_, p, _ := u.do("GET", "/api/vc/puzzle", nil)
		if !strings.HasPrefix(p["bg"].(string), "data:image/svg+xml;base64,") {
			t.Fatal("no puzzle image")
		}
		if strings.Contains(fmt.Sprint(p), "answer") {
			t.Fatal("answer leaked")
		}
		id := p["id"].(string)
		vc.mu.Lock()
		ans := vc.m[id].answer
		vc.mu.Unlock()
		return id, ans
	}
	id, ans := puzzle()
	lin := map[string]any{"id": id, "x": ans, "dur": 380}
	var ltr [][3]float64
	for i := 0; i < 20; i++ {
		ltr = append(ltr, [3]float64{float64(i * 20), float64(ans) * float64(i) / 19, 0})
	}
	lin["track"] = ltr
	code, _, _ = u.do("POST", "/api/vc/puzzle", lin)
	eq(t, code, 422, "linear bot rejected")
	id, ans = puzzle()
	h := human(float64(ans + 40))
	h["id"] = id
	code, _, _ = u.do("POST", "/api/vc/puzzle", h)
	eq(t, code, 422, "wrong place rejected")
	id, ans = puzzle()
	h = human(float64(ans + 3))
	h["id"] = id
	code, m, _ = u.do("POST", "/api/vc/puzzle", h)
	eq(t, code, 200, "human accepted")
	tok := m["token"].(string)
	code, _, _ = u.do("POST", "/api/vc/puzzle", h)
	eq(t, code, 410, "puzzle single use")

	// ثبت‌نام با کد (فرم واقعی → ریدایرکت)
	_, m, _ = u.do("POST", "/api/check-code", map[string]string{"code": acode, "captcha": tok})
	eq(t, m["ok"], true, "check-code")
	code, _, resp := u.do("POST", "/register", map[string]string{"code": acode, "username": "ali", "password": "password1", "captcha": tok})
	eq(t, code, 200, "register json")
	_ = resp
	_, m, _ = u.do("GET", "/api/me", nil)
	eq(t, m["quota"], 1000, "quota from code")
	code, _, _ = (&client{t, srv, map[string]string{}}).do("POST", "/register", map[string]string{"code": acode, "username": "bob", "password": "password1", "captcha": tok})
	if code == 200 {
		t.Fatal("code reuse allowed")
	}

	// آپلود و سهمیه
	code, m, _ = u.do("POST", "/api/upload?name=%D8%B3%D9%86%D8%AF.txt", []byte(strings.Repeat("a", 600)))
	eq(t, code, 200, "upload ok")
	fid := m["file"].(map[string]any)["id"].(string)
	code, _, _ = u.do("POST", "/api/upload?name=big.bin", []byte(strings.Repeat("b", 600)))
	eq(t, code, 413, "quota enforced")
	code, _, _ = u.do("POST", "/api/upload?name=../../evil.txt", []byte("x"))
	eq(t, code, 200, "traversal name sanitized")
	_, m, _ = u.do("GET", "/api/files", nil)
	for _, f := range m["files"].([]any) {
		if n := f.(map[string]any)["name"].(string); strings.ContainsAny(n, "/\\") {
			t.Fatal("path in name")
		}
	}

	// دانلود + Range + سربرگ‌های امن
	code, _, resp = u.do("GET", "/f/"+fid, nil, "Range", "bytes=0-9")
	eq(t, code, 206, "range")
	if !strings.Contains(resp.Header.Get("Content-Security-Policy"), "sandbox") {
		t.Fatal("no sandbox csp on user files")
	}

	// ایزوله بودن کاربران: ادمین هم فایل ali را نمی‌بیند
	code, _, _ = admin.do("GET", "/f/"+fid, nil)
	eq(t, code, 404, "admin cannot read others' files")
	code, _, _ = u.do("GET", "/api/admin/users", nil)
	eq(t, code, 403, "user not admin")
	code, _, _ = (&client{t, srv, map[string]string{}}).do("GET", "/f/"+fid, nil)
	eq(t, code, 401, "anon download")

	// تامبنیل: JPEG با EXIF جهت ۶ (چرخش ۹۰°) → خروجی عمودی
	img := image.NewRGBA(image.Rect(0, 0, 800, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 800; x++ {
			img.Set(x, y, color.RGBA{uint8(x / 4), uint8(y / 2), 90, 255})
		}
	}
	var jb bytes.Buffer
	_ = jpeg.Encode(&jb, img, nil)
	raw := jb.Bytes()
	exif := []byte{0xFF, 0xE1, 0, 0x22, 'E', 'x', 'i', 'f', 0, 0, 'M', 'M', 0, 0x2A, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 6, 0, 0, 0, 0, 0, 0}
	binary.BigEndian.PutUint16(exif[2:], uint16(len(exif)-2))
	withExif := append(append(append([]byte{}, raw[:2]...), exif...), raw[2:]...)
	// quota کاربر پایین است؛ برای این تست از حساب ادمین استفاده می‌کنیم
	code, m, _ = admin.do("POST", "/api/upload?name=photo.jpg", withExif)
	eq(t, code, 200, "photo upload")
	pid := m["file"].(map[string]any)["id"].(string)
	code, _, resp = admin.do("GET", "/t/"+pid, nil)
	eq(t, code, 200, "thumb")
	tb, _, err := image.Decode(bytes.NewReader(mustRead(resp)))
	_ = tb
	_ = err
	thumbPath := a.dir + "/thumbs/" + pid + ".jpg"
	cfg := decodeCfg(t, thumbPath)
	if !(cfg.Height > cfg.Width) {
		t.Fatalf("exif rotation not applied: %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.Height > thumbMax {
		t.Fatalf("thumb too big %d", cfg.Height)
	}

	// پوشه‌ها
	admin.do("POST", "/api/folders", map[string]string{"name": "اداری"})
	_, m, _ = admin.do("GET", "/api/folders", nil)
	fo := m["folders"].([]any)[0].(map[string]any)["id"].(string)
	admin.do("POST", "/api/files/move", map[string]any{"ids": []string{pid}, "folder": fo})
	_, m, _ = admin.do("GET", "/api/files?folder="+fo, nil)
	eq(t, len(m["files"].([]any)), 1, "folder filter")

	// حذف
	_, m, _ = admin.do("POST", "/api/files/delete", map[string]any{"ids": []string{pid}})
	eq(t, m["deleted"], 1, "deleted")
	code, _, _ = admin.do("GET", "/f/"+pid, nil)
	eq(t, code, 404, "gone")
	code, _, _ = u.do("POST", "/api/files/delete", map[string]any{"ids": []string{fid}})
	eq(t, code, 200, "user deletes own")

	// قفل brute force: ۶ رمز اشتباه → ۴۲۹ حتی با رمز درست
	_ = tok
	a.db.mu.Lock()
	a.db.CentralOff = true // حالت لوکال: کپچا خاموش
	a.db.mu.Unlock()
	atk := &client{t, srv, map[string]string{}}
	for i := 0; i < 6; i++ {
		c, _, _ := atk.do("POST", "/login", map[string]string{"username": "ali", "password": "wrong-pass"})
		if c != 401 {
			t.Fatalf("attempt %d got %d", i, c)
		}
	}
	code, _, _ = atk.do("POST", "/login", map[string]string{"username": "ali", "password": "password1"})
	eq(t, code, 429, "locked after brute force")
}

func TestGuards(t *testing.T) {
	a, srv := newApp(t)
	c := &client{t, srv, map[string]string{}}
	code, _, _ := c.do("GET", "/api/state", nil, "Host", "evil.example.com")
	_ = code
	req := httptest.NewRequest("GET", "http://evil.example.com/api/state", nil)
	req.RemoteAddr = "192.168.1.5:5555"
	rec := httptest.NewRecorder()
	a.routes().ServeHTTP(rec, req)
	eq(t, rec.Code, 421, "dns rebinding host blocked")
	req = httptest.NewRequest("GET", "http://avand.local/api/state", nil)
	req.RemoteAddr = "8.8.8.8:5555"
	rec = httptest.NewRecorder()
	a.routes().ServeHTTP(rec, req)
	eq(t, rec.Code, 403, "public IP blocked")
	req = httptest.NewRequest("GET", "http://avand.local/api/state", nil)
	req.RemoteAddr = "192.168.1.5:5555"
	rec = httptest.NewRecorder()
	a.routes().ServeHTTP(rec, req)
	eq(t, rec.Code, 200, "LAN + avand.local ok")
	// setup فقط از localhost
	req = httptest.NewRequest("POST", "http://192.168.1.9/setup", strings.NewReader(`{"username":"hack","password":"password1"}`))
	req.RemoteAddr = "192.168.1.5:5555"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://192.168.1.9")
	rec = httptest.NewRecorder()
	a.routes().ServeHTTP(rec, req)
	eq(t, rec.Code, 403, "remote setup blocked")
	// POST بدون Origin / با Origin بیگانه
	code, _, _ = c.do("POST", "/api/logout", nil, "Origin", "")
	eq(t, code, 403, "no origin")
	code, _, _ = c.do("POST", "/api/logout", nil, "Origin", "http://evil.com")
	eq(t, code, 403, "foreign origin")
}

func TestMDNS(t *testing.T) {
	q := []byte{0x12, 0x34, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0}
	for _, p := range []string{"Avand", "local"} {
		q = append(q, byte(len(p)))
		q = append(q, p...)
	}
	q = append(q, 0, 0, 1, 0x80, 1) // A, IN + QU
	id, uni, ok := matchMDNSQuery(q, "avand.local")
	if !ok || id != 0x1234 || !uni {
		t.Fatalf("query not matched %v %v %v", id, uni, ok)
	}
	if _, _, ok := matchMDNSQuery(q, "other.local"); ok {
		t.Fatal("matched wrong name")
	}
	ans := buildMDNSAnswer("avand.local", net.IPv4(192, 168, 1, 20), 0, false)
	if !bytes.HasSuffix(ans, []byte{192, 168, 1, 20}) || binary.BigEndian.Uint16(ans[6:]) != 1 {
		t.Fatalf("bad answer % x", ans)
	}
}

func TestUpdaterSignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	bin := []byte("fake-binary-contents")
	sum := sha256.Sum256(bin)
	good := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, releaseMessage("v9.0.0", hex.EncodeToString(sum[:]))))
	other, opriv, _ := ed25519.GenerateKey(nil)
	_ = other
	bad := base64.StdEncoding.EncodeToString(ed25519.Sign(opriv, releaseMessage("v9.0.0", hex.EncodeToString(sum[:]))))
	sigNow := good
	var mux = http.NewServeMux()
	var base string
	mux.HandleFunc("/repos/me/avand/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		name := "avand-" + goos() + "-" + goarch() + exe()
		fmt.Fprintf(w, `{"tag_name":"v9.0.0","assets":[{"name":%q,"browser_download_url":%q},{"name":%q,"browser_download_url":%q}]}`, name, base+"/bin", name+".sig", base+"/sig")
	})
	mux.HandleFunc("/bin", func(w http.ResponseWriter, r *http.Request) { w.Write(bin) })
	mux.HandleFunc("/sig", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(sigNow)) })
	gh := httptest.NewServer(mux)
	defer gh.Close()
	base = gh.URL
	t.Setenv("AVAND_UPDATE_API", gh.URL)
	a, _ := newApp(t)
	UpdateRepo, UpdatePubKey, Version = "me/avand", hex.EncodeToString(pub), "1.0.0"
	defer func() { UpdateRepo, UpdatePubKey, Version = "", "", "dev" }()
	if err := a.doUpdate(false); err != nil {
		t.Fatalf("valid release rejected: %v", err)
	}
	sigNow = bad
	err := a.doUpdate(false)
	if err == nil || !strings.Contains(err.Error(), "امضا") {
		t.Fatalf("forged signature accepted: %v", err)
	}
	if newer("1.0.0", "1.0.0") || !newer("v1.2.0", "1.1.9") || newer("0.9.9", "1.0.0") {
		t.Fatal("semver compare")
	}
}
