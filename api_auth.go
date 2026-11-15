package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// بدنه‌ی JSON یا فرم (فرم واقعی = ذخیره‌ی رمز در مرورگر)
func readBody(w http.ResponseWriter, r *http.Request) (map[string]string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	out := map[string]string{}
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var m map[string]any
		if json.NewDecoder(r.Body).Decode(&m) != nil {
			return nil, false
		}
		for k, v := range m {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
		return out, false
	}
	if r.ParseForm() != nil {
		return nil, true
	}
	for k := range r.PostForm {
		out[k] = r.PostForm.Get(k)
	}
	return out, true
}

// پاسخ یکسان برای فرم (ریدایرکت) و JSON
func reply(w http.ResponseWriter, r *http.Request, isForm bool, ok bool, status int, msg, code, tab, dest string) {
	if isForm {
		if ok {
			http.Redirect(w, r, dest, http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/?e="+url.QueryEscape(msg)+"&t="+tab, http.StatusSeeOther)
		return
	}
	if ok {
		writeJSON(w, 200, map[string]any{"ok": true})
		return
	}
	failCode(w, status, msg, code)
}

func (a *App) state(w http.ResponseWriter, r *http.Request) {
	a.db.mu.RLock()
	none := len(a.db.Users) == 0
	a.db.mu.RUnlock()
	writeJSON(w, 200, map[string]any{"ok": true, "setup": none, "local": isLoopback(r), "central": a.central()})
}

func (a *App) checkCode(w http.ResponseWriter, r *http.Request) {
	b, _ := readBody(w, r)
	if b == nil {
		fail(w, 400, "درخواست نامعتبر")
		return
	}
	ip := clientKey(r)
	if lim.blocked("ip:"+ip, 10, 900) || lim.blocked("all:code", 200, 3600) {
		fail(w, 429, "تلاش‌های ناموفق زیاد بود. چند دقیقه‌ی دیگر دوباره امتحان کنید.")
		return
	}
	if code, msg := a.needCaptcha(r, b["captcha"]); code != "" {
		failCode(w, 403, msg, code)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(b["code"]))
	a.db.mu.RLock()
	ok := false
	for _, c := range a.db.Codes {
		if c.UsedAt == 0 && safeEq(c.Code, code) {
			ok = true
		}
	}
	a.db.mu.RUnlock()
	if !ok {
		lim.hit("ip:" + ip)
		lim.hit("all:code")
		fail(w, 404, "کد دسترسی نامعتبر است یا قبلاً استفاده شده است.")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	b, isForm := readBody(w, r)
	if b == nil {
		fail(w, 400, "درخواست نامعتبر")
		return
	}
	bad := func(st int, msg, code string) { reply(w, r, isForm, false, st, msg, code, "access", "") }
	ip := clientKey(r)
	if lim.blocked("ip:"+ip, 10, 900) || lim.blocked("all:code", 200, 3600) {
		bad(429, "تلاش‌های ناموفق زیاد بود. چند دقیقه‌ی دیگر دوباره امتحان کنید.", "")
		return
	}
	if code, msg := a.needCaptcha(r, b["captcha"]); code != "" {
		bad(403, msg, code)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(b["code"]))
	name, pw := strings.TrimSpace(b["username"]), b["password"]
	if !validUsername(name) {
		bad(400, "نام کاربری باید ۳ تا ۲۴ نویسه و فقط شامل حروف انگلیسی، عدد، _ و . باشد.", "")
		return
	}
	if !validPassword(pw) {
		bad(400, "رمز عبور باید حداقل ۸ نویسه باشد.", "")
		return
	}
	if b["password2"] != "" && b["password2"] != pw {
		bad(400, "رمز عبور و تکرارش یکسان نیست.", "")
		return
	}
	salt := randBytes(16)
	hash := hashPassword(pw, salt, pbkdf2Iter) // خارج از قفل

	a.db.mu.Lock()
	var c *Code
	for _, x := range a.db.Codes {
		if x.UsedAt == 0 && safeEq(x.Code, code) {
			c = x
		}
	}
	if c == nil {
		a.db.mu.Unlock()
		lim.hit("ip:" + ip)
		lim.hit("all:code")
		bad(404, "کد دسترسی نامعتبر است یا قبلاً استفاده شده است.", "")
		return
	}
	if a.db.userByName(name) != nil {
		a.db.mu.Unlock()
		bad(409, "این نام کاربری قبلاً گرفته شده است.", "")
		return
	}
	u := &User{ID: randHex(8), Name: name, Hash: hash, Salt: encB64(salt), Iter: pbkdf2Iter, Admin: c.Admin, Active: true, Quota: c.Quota, Created: now()}
	a.db.Users = append(a.db.Users, u)
	c.UsedAt, c.UsedBy = now(), u.ID
	a.db.touch()
	a.db.mu.Unlock()
	a.newSession(w, r, u.ID)
	reply(w, r, isForm, true, 200, "", "", "", "/app")
}

var dummySalt = make([]byte, 16)

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	b, isForm := readBody(w, r)
	if b == nil {
		fail(w, 400, "درخواست نامعتبر")
		return
	}
	bad := func(st int, msg, code string) { reply(w, r, isForm, false, st, msg, code, "member", "") }
	name, pw := strings.TrimSpace(b["username"]), b["password"]
	if len(pw) > 128 {
		pw = pw[:128]
	}
	ip, ukey := clientKey(r), "user:"+strings.ToLower(name)
	if len(ukey) > 60 {
		ukey = ukey[:60]
	}
	if lim.blocked("ip:"+ip, 10, 900) || lim.blocked(ukey, 6, 900) || lim.blocked("all:login", 300, 3600) {
		bad(429, "تلاش‌های ناموفق زیاد بود. چند دقیقه‌ی دیگر دوباره امتحان کنید.", "")
		return
	}
	if code, msg := a.needCaptcha(r, b["captcha"]); code != "" {
		bad(403, msg, code)
		return
	}
	a.db.mu.RLock()
	u := a.db.userByName(name)
	var cp User
	if u != nil {
		cp = *u
	}
	a.db.mu.RUnlock()
	salt, iter, want := dummySalt, pbkdf2Iter, ""
	if u != nil {
		salt, iter, want = decB64(cp.Salt), cp.Iter, cp.Hash
	}
	got := hashPassword(pw, salt, iter) // حتی اگر کاربر نباشد حساب می‌شود
	if u == nil || !safeEq(got, want) {
		lim.hit("ip:" + ip)
		lim.hit(ukey)
		lim.hit("all:login")
		bad(401, "رمز عبور یا نام کاربری اشتباه است.", "")
		return
	}
	if !cp.Active {
		bad(403, "این حساب توسط مدیر غیرفعال شده است.", "")
		return
	}
	a.newSession(w, r, cp.ID)
	reply(w, r, isForm, true, 200, "", "", "", "/app")
}

// ساخت اولین مدیر؛ فقط روی خود کامپیوتر میزبان (localhost) و فقط وقتی هنوز کاربری نیست
func (a *App) setup(w http.ResponseWriter, r *http.Request) {
	b, isForm := readBody(w, r)
	if b == nil {
		fail(w, 400, "درخواست نامعتبر")
		return
	}
	bad := func(st int, msg string) { reply(w, r, isForm, false, st, msg, "", "setup", "") }
	if !isLoopback(r) {
		bad(403, "راه‌اندازی اولیه فقط روی خود کامپیوتر میزبان ممکن است.")
		return
	}
	name, pw := strings.TrimSpace(b["username"]), b["password"]
	if !validUsername(name) {
		bad(400, "نام کاربری باید ۳ تا ۲۴ نویسه و فقط شامل حروف انگلیسی، عدد، _ و . باشد.")
		return
	}
	if !validPassword(pw) {
		bad(400, "رمز عبور باید حداقل ۸ نویسه باشد.")
		return
	}
	if b["password2"] != "" && b["password2"] != pw {
		bad(400, "رمز عبور و تکرارش یکسان نیست.")
		return
	}
	salt := randBytes(16)
	hash := hashPassword(pw, salt, pbkdf2Iter)
	a.db.mu.Lock()
	if len(a.db.Users) > 0 {
		a.db.mu.Unlock()
		bad(409, "راه‌اندازی قبلاً انجام شده است.")
		return
	}
	u := &User{ID: randHex(8), Name: name, Hash: hash, Salt: encB64(salt), Iter: pbkdf2Iter, Admin: true, Active: true, Created: now()}
	a.db.Users = append(a.db.Users, u)
	a.db.touch()
	a.db.mu.Unlock()
	a.newSession(w, r, u.ID)
	reply(w, r, isForm, true, 200, "", "", "", "/app")
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	a.clearSession(w, r)
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) me(w http.ResponseWriter, r *http.Request) {
	u := a.needUser(w, r)
	if u == nil {
		return
	}
	a.db.mu.RLock()
	used := a.db.usedBytes(u.ID)
	n := 0
	for _, f := range a.db.Files {
		if f.Owner == u.ID {
			n++
		}
	}
	a.db.mu.RUnlock()
	writeJSON(w, 200, map[string]any{"ok": true, "username": u.Name, "admin": u.Admin, "quota": u.Quota, "used": used, "count": n, "urls": a.urls})
}

// تغییر رمز: رمز فعلی لازم است؛ بقیه‌ی نشست‌های همین کاربر بسته می‌شود
func (a *App) changePassword(w http.ResponseWriter, r *http.Request) {
	u := a.needUser(w, r)
	if u == nil {
		return
	}
	b, _ := readBody(w, r)
	if b == nil {
		fail(w, 400, "درخواست نامعتبر")
		return
	}
	key := "pw:" + u.ID
	if lim.blocked(key, 6, 900) {
		fail(w, 429, "تلاش‌ها زیاد بود. چند دقیقه‌ی دیگر دوباره امتحان کنید.")
		return
	}
	if !safeEq(hashPassword(b["current"], decB64(u.Salt), u.Iter), u.Hash) {
		lim.hit(key)
		fail(w, 403, "رمز فعلی اشتباه است.")
		return
	}
	if !validPassword(b["new"]) {
		fail(w, 400, "رمز جدید باید حداقل ۸ نویسه باشد.")
		return
	}
	salt := randBytes(16)
	hash := hashPassword(b["new"], salt, pbkdf2Iter)
	keepHash := ""
	if c, err := r.Cookie(sessionCookie); err == nil {
		keepHash = sha256hex(c.Value)
	}
	a.db.mu.Lock()
	if x := a.db.userByID(u.ID); x != nil {
		x.Hash, x.Salt, x.Iter = hash, encB64(salt), pbkdf2Iter
	}
	keep := a.db.Session[:0]
	for _, s := range a.db.Session {
		if s.UserID != u.ID || s.Hash == keepHash {
			keep = append(keep, s)
		}
	}
	a.db.Session = keep
	a.db.touch()
	a.db.mu.Unlock()
	writeJSON(w, 200, map[string]any{"ok": true})
}
