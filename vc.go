package main

// VoidCaptcha: بررسی مرورگر (اثبات کار) + پازل انسان‌سنج؛ همه سمت سرور راستی‌آزمایی می‌شود.

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	powBits     = 17
	bcCookie    = "avand_bc"
	bcTTL       = 12 * time.Hour
	puzW, puzH  = 320, 160
	puzzleShape = "M0 0H16a6 6 0 1 1 12 0H44V16a6 6 0 1 1 0 12V44H0Z"
)

type vcEntry struct {
	kind    string // puz | pow | pass
	answer  int
	seed    string
	uses    int
	created int64
}

type vcStore struct {
	mu sync.Mutex
	m  map[string]*vcEntry
	bc map[string]int64 // sha256(token) → انقضا
}

var vc = &vcStore{m: map[string]*vcEntry{}, bc: map[string]int64{}}

func (s *vcStore) gc() {
	cut := now() - 3600
	for k, v := range s.m {
		if v.created < cut {
			delete(s.m, k)
		}
	}
	for k, v := range s.bc {
		if v < now() {
			delete(s.bc, k)
		}
	}
}

func randInt(min, max int) int {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return min + int(binary.BigEndian.Uint32(b))%(max-min+1)
}

func clientKey(r *http.Request) string { return remoteIP(r).String() }

func bcValid(r *http.Request) bool {
	c, err := r.Cookie(bcCookie)
	if err != nil {
		return false
	}
	vc.mu.Lock()
	defer vc.mu.Unlock()
	exp, ok := vc.bc[sha256hex(c.Value)]
	return ok && exp > now()
}

// قبل از ورود/ثبت‌نام: هم «بررسی مرورگر» و هم توکن پازل (۵ دقیقه، حداکثر ۳ بار مصرف)
func (a *App) needCaptcha(r *http.Request, token string) (code, msg string) {
	if !a.central() {
		return "", "" // حالت لوکال: کپچا خاموش
	}
	if !bcValid(r) {
		return "bc", "بررسی مرورگر لازم است. صفحه را دوباره باز کنید."
	}
	vc.mu.Lock()
	defer vc.mu.Unlock()
	e := vc.m[token]
	if token == "" || e == nil || e.kind != "pass" || e.uses >= 3 || now()-e.created > 300 {
		return "captcha", "تأیید VoidCaptcha انجام نشده یا منقضی شده است."
	}
	e.uses++
	return "", ""
}

func leadingZeroBits(sum [32]byte) int {
	n := 0
	for _, b := range sum {
		if b == 0 {
			n += 8
			continue
		}
		for m := byte(0x80); m != 0 && b&m == 0; m >>= 1 {
			n++
		}
		break
	}
	return n
}

// GET /api/vc/browser
func (a *App) vcBrowserGet(w http.ResponseWriter, r *http.Request) {
	if !a.central() || bcValid(r) {
		writeJSON(w, 200, map[string]any{"ok": true, "valid": true})
		return
	}
	k := "vcnew:" + clientKey(r)
	if lim.blocked(k, 40, 900) {
		fail(w, 429, "تلاش‌ها زیاد بود. کمی بعد دوباره امتحان کنید.")
		return
	}
	lim.hit(k)
	id, seed := randHex(16), randHex(12)
	vc.mu.Lock()
	vc.gc()
	vc.m[id] = &vcEntry{kind: "pow", answer: powBits, seed: seed, created: now()}
	vc.mu.Unlock()
	writeJSON(w, 200, map[string]any{"ok": true, "valid": false, "id": id, "seed": seed, "bits": powBits})
}

var headlessRe = regexp.MustCompile(`(?i)HeadlessChrome|PhantomJS|Selenium|puppeteer|playwright`)

// POST /api/vc/browser {id, nonce, sig:{wd}}
func (a *App) vcBrowserPost(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ID    string         `json:"id"`
		Nonce string         `json:"nonce"`
		Sig   map[string]any `json:"sig"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&b) != nil {
		fail(w, 400, "درخواست نامعتبر")
		return
	}
	vc.mu.Lock()
	e := vc.m[b.ID]
	if e != nil && e.kind == "pow" {
		delete(vc.m, b.ID) // یک‌بارمصرف
	} else {
		e = nil
	}
	vc.mu.Unlock()
	if e == nil || now()-e.created > 300 {
		fail(w, 410, "چالش منقضی شده؛ دوباره تلاش کنید.")
		return
	}
	if _, err := strconv.ParseUint(b.Nonce, 10, 64); err != nil || len(b.Nonce) > 20 {
		fail(w, 400, "پاسخ نامعتبر")
		return
	}
	if leadingZeroBits(sha256.Sum256([]byte(e.seed+":"+b.Nonce))) < e.answer {
		fail(w, 403, "بررسی مرورگر ناموفق بود.")
		return
	}
	if wd, _ := b.Sig["wd"].(bool); wd || headlessRe.MatchString(r.UserAgent()) {
		fail(w, 403, "مرورگر خودکار شناسایی شد.")
		return
	}
	tok := randHex(32)
	vc.mu.Lock()
	vc.bc[sha256hex(tok)] = now() + int64(bcTTL.Seconds())
	vc.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: bcCookie, Value: tok, Path: "/", MaxAge: int(bcTTL.Seconds()), HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil})
	writeJSON(w, 200, map[string]any{"ok": true})
}

func artwork() string {
	h := randInt(0, 359)
	var s strings.Builder
	fmt.Fprintf(&s, `<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="hsl(%d,70%%,38%%)"/><stop offset="1" stop-color="hsl(%d,75%%,18%%)"/></linearGradient>`+
		`<filter id="n"><feTurbulence type="fractalNoise" baseFrequency="%.2f" numOctaves="2" seed="%d"/><feColorMatrix values="0 0 0 0 1  0 0 0 0 1  0 0 0 0 1  0 0 0 .22 0"/></filter></defs>`+
		`<rect width="%d" height="%d" fill="url(#g)"/>`, h, (h+randInt(40, 120))%360, float64(randInt(6, 14))/100, randInt(1, 999), puzW, puzH)
	for i := 0; i < 16; i++ {
		fmt.Fprintf(&s, `<circle cx="%d" cy="%d" r="%d" fill="hsl(%d,%d%%,%d%%)" fill-opacity="%.2f"/>`,
			randInt(0, puzW), randInt(0, puzH), randInt(8, 46), (h+randInt(0, 200))%360, randInt(50, 90), randInt(30, 65), float64(randInt(18, 55))/100)
	}
	for i := 0; i < 9; i++ {
		fmt.Fprintf(&s, `<polygon points="%d,%d %d,%d %d,%d" fill="hsl(%d,%d%%,%d%%)" fill-opacity="%.2f"/>`,
			randInt(0, puzW), randInt(0, puzH), randInt(0, puzW), randInt(0, puzH), randInt(0, puzW), randInt(0, puzH),
			(h+randInt(0, 260))%360, randInt(50, 90), randInt(25, 70), float64(randInt(15, 45))/100)
	}
	for i := 0; i < 6; i++ {
		fmt.Fprintf(&s, `<path d="M0 %d Q %d %d %d %d" stroke="hsl(%d,80%%,70%%)" stroke-opacity=".35" stroke-width="%d" fill="none"/>`,
			randInt(0, puzH), randInt(60, 260), randInt(0, puzH), puzW, randInt(0, puzH), (h+randInt(0, 300))%360, randInt(1, 4))
	}
	fmt.Fprintf(&s, `<rect width="%d" height="%d" filter="url(#n)"/>`, puzW, puzH)
	return s.String()
}

func dataSVG(s string) string {
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(s))
}

func (a *App) vcPuzzleGuard(w http.ResponseWriter, r *http.Request) bool {
	ip := clientKey(r)
	if lim.blocked("vcnew:"+ip, 60, 900) || lim.blocked("vcfail:"+ip, 15, 900) {
		fail(w, 429, "تلاش‌ها زیاد بود. کمی بعد دوباره امتحان کنید.")
		return false
	}
	return true
}

// GET /api/vc/puzzle: جواب فقط سمت سرور می‌ماند
func (a *App) vcPuzzleGet(w http.ResponseWriter, r *http.Request) {
	if !a.vcPuzzleGuard(w, r) {
		return
	}
	lim.hit("vcnew:" + clientKey(r))
	tx, ty := randInt(110, puzW-64), randInt(34, puzH-66)
	art := artwork()
	var dx int
	if tx > puzW/2 {
		dx = randInt(40, tx-70)
	} else {
		dx = randInt(tx+70, puzW-64)
	}
	dy := randInt(24, puzH-64)
	bg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">%s`+
		`<path d="%s" transform="translate(%d %d) rotate(%d 22 22)" fill="none" stroke="#fff" stroke-opacity=".28" stroke-width="1.5" stroke-dasharray="4 3"/>`+
		`<path d="%s" transform="translate(%d %d)" fill="#000" fill-opacity=".55" stroke="#fff" stroke-opacity=".85" stroke-width="1.5"/></svg>`,
		puzW, puzH, puzW, puzH, art, puzzleShape, dx, dy, randInt(0, 3)*90, puzzleShape, tx, ty)
	piece := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="60" height="60" viewBox="-5 -8 60 60"><clipPath id="c"><path d="%s"/></clipPath>`+
		`<g clip-path="url(#c)"><g transform="translate(%d %d)">%s</g></g>`+
		`<path d="%s" fill="none" stroke="#000" stroke-opacity=".35" stroke-width="3"/><path d="%s" fill="none" stroke="#fff" stroke-opacity=".95" stroke-width="1.5"/></svg>`,
		puzzleShape, -tx, -ty, art, puzzleShape, puzzleShape)
	id := randHex(16)
	vc.mu.Lock()
	vc.gc()
	vc.m[id] = &vcEntry{kind: "puz", answer: tx, created: now()}
	vc.mu.Unlock()
	writeJSON(w, 200, map[string]any{"ok": true, "id": id, "w": puzW, "h": puzH, "ty": ty, "bg": dataSVG(bg), "piece": dataSVG(piece)})
}

func std(a []float64) float64 {
	var m float64
	for _, x := range a {
		m += x
	}
	m /= float64(len(a))
	var v float64
	for _, x := range a {
		v += (x - m) * (x - m)
	}
	return math.Sqrt(v / float64(len(a)))
}

// حرکت کشیدن باید شبیه انسان باشد: نقاط کافی، زمان‌بندی طبیعی، سرعت متغیر
func humanLike(track [][3]float64, dur float64) bool {
	if len(track) < 6 || len(track) > 400 {
		return false
	}
	for i := 1; i < len(track); i++ {
		if track[i][0] < track[i-1][0] {
			return false
		}
	}
	span := track[len(track)-1][0] - track[0][0]
	if dur < 350 || dur > 40000 || span < 250 || math.Abs(span-dur) > 2500 {
		return false
	}
	xs := map[float64]bool{}
	for _, p := range track {
		xs[p[1]] = true
	}
	if len(xs) < 5 {
		return false
	}
	speeds := make([]float64, 0, len(track))
	ys := make([]float64, 0, len(track))
	for i, p := range track {
		ys = append(ys, p[2])
		if i > 0 {
			dt := math.Max(p[0]-track[i-1][0], 1)
			speeds = append(speeds, math.Abs(p[1]-track[i-1][1])/dt)
		}
	}
	var mean float64
	for _, s := range speeds {
		mean += s
	}
	mean /= float64(len(speeds))
	cv := 0.0
	if mean > 0 {
		cv = std(speeds) / mean
	}
	return cv > 0.18 || std(ys) > 0.4
}

// POST /api/vc/puzzle {id, x, track, dur} → {token}
func (a *App) vcPuzzlePost(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ID    string       `json:"id"`
		X     float64      `json:"x"`
		Track [][3]float64 `json:"track"`
		Dur   float64      `json:"dur"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&b) != nil {
		fail(w, 400, "درخواست نامعتبر")
		return
	}
	if !a.vcPuzzleGuard(w, r) {
		return
	}
	vc.mu.Lock()
	e := vc.m[b.ID]
	if e != nil && e.kind == "puz" {
		delete(vc.m, b.ID) // هر پازل فقط یک تلاش
	} else {
		e = nil
	}
	vc.mu.Unlock()
	if e == nil {
		fail(w, 410, "پازل منقضی شده؛ پازل جدید بگیرید.")
		return
	}
	if now()-e.created > 180 || math.Abs(b.X-float64(e.answer)) > 5 || !humanLike(b.Track, b.Dur) {
		lim.hit("vcfail:" + clientKey(r))
		fail(w, 422, "تأیید نشد. دوباره تلاش کنید.")
		return
	}
	tok := randHex(24)
	vc.mu.Lock()
	vc.m[tok] = &vcEntry{kind: "pass", created: now()}
	vc.mu.Unlock()
	writeJSON(w, 200, map[string]any{"ok": true, "token": tok})
}
