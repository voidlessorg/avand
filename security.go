package main

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	sessionCookie = "avand_session"
	sessionTTL    = 30 * 24 * time.Hour
	pbkdf2Iter    = 210000
)

func now() int64 { return time.Now().Unix() }

func randBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}
func randHex(n int) string { return hex.EncodeToString(randBytes(n)) }

func sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func equalFold(a, b string) bool { return strings.EqualFold(a, b) }

func hashPassword(pw string, salt []byte, iter int) string {
	k, _ := pbkdf2.Key(sha256.New, pw, salt, iter, 32)
	return base64.StdEncoding.EncodeToString(k)
}

func safeEq(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

// ---------- محدودیت تلاش (در حافظه) ----------
type limiter struct {
	mu sync.Mutex
	m  map[string][]int64
}

var lim = &limiter{m: map[string][]int64{}}

func (l *limiter) count(key string, window int64) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := now() - window
	old := l.m[key]
	keep := old[:0]
	for _, t := range old {
		if t > cut {
			keep = append(keep, t)
		}
	}
	if len(keep) == 0 {
		delete(l.m, key)
	} else {
		l.m[key] = keep
	}
	return len(keep)
}
func (l *limiter) hit(key string) {
	l.mu.Lock()
	l.m[key] = append(l.m[key], now())
	l.mu.Unlock()
}
func (l *limiter) blocked(key string, max int, window int64) bool { return l.count(key, window) >= max }

// ---------- شبکه: فقط LAN ----------
func remoteIP(r *http.Request) net.IP {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		h = r.RemoteAddr
	}
	return net.ParseIP(h)
}

func isPrivate(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func isLoopback(r *http.Request) bool {
	ip := remoteIP(r)
	return ip != nil && ip.IsLoopback()
}

// Host مجاز: localhost، آدرس‌های IP خصوصی، avand.local و نام کامپیوتر (ضد DNS-rebinding)
func (a *App) hostAllowed(host string) bool {
	h := strings.ToLower(host)
	if i := strings.LastIndex(h, ":"); i > 0 && !strings.Contains(h[i:], "]") {
		h = h[:i]
	}
	h = strings.Trim(h, "[]")
	if h == "localhost" || h == a.cfg.Host+".local" || (a.hostname != "" && h == strings.ToLower(a.hostname)) || h == strings.ToLower(a.hostname)+".local" {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return isPrivate(ip)
	}
	return false
}

func (a *App) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isPrivate(remoteIP(r)) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !a.hostAllowed(r.Host) {
			http.Error(w, "bad host", http.StatusMisdirectedRequest)
			return
		}
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/" || r.URL.Path == "/app" || r.URL.Path == "/admin" {
			h.Set("Cache-Control", "no-store")
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			o := r.Header.Get("Origin")
			if o == "" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			u, err := url.Parse(o)
			if err != nil || !strings.EqualFold(u.Host, r.Host) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- پاسخ‌ها ----------
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": msg})
}
func failCode(w http.ResponseWriter, status int, msg, code string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": msg, "code": code})
}

// ---------- نشست ----------
func (a *App) newSession(w http.ResponseWriter, r *http.Request, userID string) {
	tok := base64.RawURLEncoding.EncodeToString(randBytes(32))
	a.db.mu.Lock()
	a.db.Session = append(a.db.Session, &Session{Hash: sha256hex(tok), UserID: userID, Expires: now() + int64(sessionTTL.Seconds())})
	keep := a.db.Session[:0]
	for _, s := range a.db.Session {
		if s.Expires > now() {
			keep = append(keep, s)
		}
	}
	a.db.Session = keep
	a.db.touch()
	a.db.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: tok, Path: "/", MaxAge: int(sessionTTL.Seconds()),
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil})
}

func (a *App) clearSession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		h := sha256hex(c.Value)
		a.db.mu.Lock()
		keep := a.db.Session[:0]
		for _, s := range a.db.Session {
			if s.Hash != h {
				keep = append(keep, s)
			}
		}
		a.db.Session = keep
		a.db.touch()
		a.db.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
}

// کاربر واردشده (کپی مستقل از ساختار تا بدون قفل قابل استفاده باشد)
func (a *App) currentUser(r *http.Request) *User {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	h := sha256hex(c.Value)
	a.db.mu.RLock()
	defer a.db.mu.RUnlock()
	for _, s := range a.db.Session {
		if safeEq(s.Hash, h) && s.Expires > now() {
			if u := a.db.userByID(s.UserID); u != nil && u.Active {
				cp := *u
				return &cp
			}
		}
	}
	return nil
}

func (a *App) needUser(w http.ResponseWriter, r *http.Request) *User {
	u := a.currentUser(r)
	if u == nil {
		fail(w, http.StatusUnauthorized, "وارد نشده‌اید.")
	}
	return u
}

func (a *App) needAdmin(w http.ResponseWriter, r *http.Request) *User {
	u := a.needUser(w, r)
	if u == nil {
		return nil
	}
	if !u.Admin {
		fail(w, http.StatusForbidden, "دسترسی ندارید.")
		return nil
	}
	return u
}

// ---------- اعتبارسنجی ----------
func validUsername(s string) bool {
	if len(s) < 3 || len(s) > 24 {
		return false
	}
	for _, c := range s {
		if !(c < unicode.MaxASCII && (unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '.')) {
			return false
		}
	}
	return true
}
func validPassword(s string) bool { return len(s) >= 8 && len(s) <= 128 }
