package main

import (
	"net/http"
	"sort"
	"strings"
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newCode() string {
	pick := func(n int) string {
		b := randBytes(n)
		out := make([]byte, n)
		for i, x := range b {
			out[i] = codeAlphabet[int(x)%len(codeAlphabet)]
		}
		return string(out)
	}
	return "AVD-" + pick(4) + "-" + pick(4)
}

func (a *App) adminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/stats", a.adminStats)
	mux.HandleFunc("GET /api/admin/users", a.adminUsers)
	mux.HandleFunc("POST /api/admin/user", a.adminUserSet)
	mux.HandleFunc("POST /api/admin/user/delete", a.adminUserDelete)
	mux.HandleFunc("POST /api/admin/user/reset", a.adminUserReset)
	mux.HandleFunc("GET /api/admin/codes", a.adminCodes)
	mux.HandleFunc("POST /api/admin/codes", a.adminCodesNew)
	mux.HandleFunc("POST /api/admin/codes/delete", a.adminCodesDel)
	mux.HandleFunc("GET /api/admin/settings", a.adminSettings)
	mux.HandleFunc("POST /api/admin/settings", a.adminSettingsSet)
	mux.HandleFunc("POST /api/admin/update/check", a.adminUpdateCheck)
	mux.HandleFunc("POST /api/admin/quit", a.adminQuit)
}

func (a *App) adminStats(w http.ResponseWriter, r *http.Request) {
	if a.needAdmin(w, r) == nil {
		return
	}
	a.db.mu.RLock()
	var total int64
	for _, f := range a.db.Files {
		total += f.Size
	}
	users, files, free := len(a.db.Users), len(a.db.Files), int64(0)
	a.db.mu.RUnlock()
	free, disk := diskFree(a.dir)
	writeJSON(w, 200, map[string]any{"ok": true, "users": users, "files": files, "bytes": total, "disk_free": free, "disk_total": disk})
}

func (a *App) adminUsers(w http.ResponseWriter, r *http.Request) {
	if a.needAdmin(w, r) == nil {
		return
	}
	type row struct {
		ID      string `json:"id"`
		Name    string `json:"username"`
		Admin   bool   `json:"admin"`
		Active  bool   `json:"active"`
		Quota   int64  `json:"quota"`
		Used    int64  `json:"used"`
		Files   int    `json:"files"`
		Created int64  `json:"created"`
	}
	a.db.mu.RLock()
	out := []row{}
	for _, u := range a.db.Users {
		x := row{ID: u.ID, Name: u.Name, Admin: u.Admin, Active: u.Active, Quota: u.Quota, Created: u.Created}
		for _, f := range a.db.Files {
			if f.Owner == u.ID {
				x.Used += f.Size
				x.Files++
			}
		}
		out = append(out, x)
	}
	a.db.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Created < out[j].Created })
	writeJSON(w, 200, map[string]any{"ok": true, "users": out})
}

// {id, quota?: بایت (۰ = نامحدود), active?: bool}
func (a *App) adminUserSet(w http.ResponseWriter, r *http.Request) {
	me := a.needAdmin(w, r)
	if me == nil {
		return
	}
	var b struct {
		ID     string `json:"id"`
		Quota  *int64 `json:"quota"`
		Active *bool  `json:"active"`
	}
	if !decodeJSON(w, r, &b) {
		return
	}
	a.db.mu.Lock()
	defer a.db.mu.Unlock()
	u := a.db.userByID(b.ID)
	if u == nil {
		fail(w, 404, "کاربر پیدا نشد.")
		return
	}
	if b.Quota != nil {
		if *b.Quota < 0 {
			fail(w, 400, "سهمیه نامعتبر است.")
			return
		}
		u.Quota = *b.Quota
	}
	if b.Active != nil {
		if !*b.Active && u.ID == me.ID {
			fail(w, 400, "حساب خودتان را نمی‌توانید غیرفعال کنید.")
			return
		}
		u.Active = *b.Active
		if !u.Active {
			keep := a.db.Session[:0]
			for _, s := range a.db.Session {
				if s.UserID != u.ID {
					keep = append(keep, s)
				}
			}
			a.db.Session = keep
		}
	}
	a.db.touch()
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) adminUserDelete(w http.ResponseWriter, r *http.Request) {
	me := a.needAdmin(w, r)
	if me == nil {
		return
	}
	var b struct{ ID string }
	if !decodeJSON(w, r, &b) {
		return
	}
	if b.ID == me.ID {
		fail(w, 400, "حساب خودتان را نمی‌توانید حذف کنید.")
		return
	}
	var gone []string
	a.db.mu.Lock()
	if a.db.userByID(b.ID) == nil {
		a.db.mu.Unlock()
		fail(w, 404, "کاربر پیدا نشد.")
		return
	}
	us := a.db.Users[:0]
	for _, u := range a.db.Users {
		if u.ID != b.ID {
			us = append(us, u)
		}
	}
	a.db.Users = us
	fs := a.db.Files[:0]
	for _, f := range a.db.Files {
		if f.Owner == b.ID {
			gone = append(gone, f.ID)
		} else {
			fs = append(fs, f)
		}
	}
	a.db.Files = fs
	fo := a.db.Folders[:0]
	for _, f := range a.db.Folders {
		if f.Owner != b.ID {
			fo = append(fo, f)
		}
	}
	a.db.Folders = fo
	ss := a.db.Session[:0]
	for _, s := range a.db.Session {
		if s.UserID != b.ID {
			ss = append(ss, s)
		}
	}
	a.db.Session = ss
	a.db.touch()
	a.db.mu.Unlock()
	for _, id := range gone {
		a.removeFileData(id)
	}
	writeJSON(w, 200, map[string]any{"ok": true, "files_deleted": len(gone)})
}

// مدیر برای کاربری که رمزش را فراموش کرده رمز جدید می‌گذارد
func (a *App) adminUserReset(w http.ResponseWriter, r *http.Request) {
	if a.needAdmin(w, r) == nil {
		return
	}
	var b struct{ ID, Password string }
	if !decodeJSON(w, r, &b) {
		return
	}
	if !validPassword(b.Password) {
		fail(w, 400, "رمز باید حداقل ۸ نویسه باشد.")
		return
	}
	salt := randBytes(16)
	hash := hashPassword(b.Password, salt, pbkdf2Iter)
	a.db.mu.Lock()
	defer a.db.mu.Unlock()
	u := a.db.userByID(b.ID)
	if u == nil {
		fail(w, 404, "کاربر پیدا نشد.")
		return
	}
	u.Hash, u.Salt, u.Iter = hash, encB64(salt), pbkdf2Iter
	ss := a.db.Session[:0]
	for _, s := range a.db.Session {
		if s.UserID != u.ID {
			ss = append(ss, s)
		}
	}
	a.db.Session = ss
	a.db.touch()
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) adminCodes(w http.ResponseWriter, r *http.Request) {
	if a.needAdmin(w, r) == nil {
		return
	}
	type row struct {
		*Code
		UsedByName string `json:"used_by_name"`
	}
	a.db.mu.RLock()
	out := []row{}
	for i := len(a.db.Codes) - 1; i >= 0; i-- {
		c := a.db.Codes[i]
		n := ""
		if u := a.db.userByID(c.UsedBy); u != nil {
			n = u.Name
		}
		out = append(out, row{c, n})
	}
	a.db.mu.RUnlock()
	writeJSON(w, 200, map[string]any{"ok": true, "codes": out})
}

// {count ≤ 100, quota (بایت؛ ۰ = نامحدود), note, admin}
func (a *App) adminCodesNew(w http.ResponseWriter, r *http.Request) {
	if a.needAdmin(w, r) == nil {
		return
	}
	var b struct {
		Count int    `json:"count"`
		Quota int64  `json:"quota"`
		Note  string `json:"note"`
		Admin bool   `json:"admin"`
	}
	if !decodeJSON(w, r, &b) {
		return
	}
	if b.Count < 1 {
		b.Count = 1
	}
	if b.Count > 100 {
		b.Count = 100
	}
	if b.Quota < 0 {
		fail(w, 400, "سهمیه نامعتبر است.")
		return
	}
	note := strings.TrimSpace(b.Note)
	if r := []rune(note); len(r) > 100 {
		note = string(r[:100])
	}
	codes := []string{}
	a.db.mu.Lock()
	for i := 0; i < b.Count; i++ {
		c := newCode()
		a.db.Codes = append(a.db.Codes, &Code{Code: c, Note: note, Quota: b.Quota, Admin: b.Admin, Created: now()})
		codes = append(codes, c)
	}
	a.db.touch()
	a.db.mu.Unlock()
	writeJSON(w, 200, map[string]any{"ok": true, "codes": codes})
}

func (a *App) adminCodesDel(w http.ResponseWriter, r *http.Request) {
	if a.needAdmin(w, r) == nil {
		return
	}
	var b struct{ Code string }
	if !decodeJSON(w, r, &b) {
		return
	}
	a.db.mu.Lock()
	defer a.db.mu.Unlock()
	keep, found := a.db.Codes[:0], false
	for _, c := range a.db.Codes {
		if c.Code == strings.ToUpper(strings.TrimSpace(b.Code)) && c.UsedAt == 0 {
			found = true
			continue
		}
		keep = append(keep, c)
	}
	a.db.Codes = keep
	if !found {
		fail(w, 404, "کد پیدا نشد یا قبلاً مصرف شده است.")
		return
	}
	a.db.touch()
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) settingsView() map[string]any {
	a.db.mu.RLock()
	off, noAuto := a.db.CentralOff, a.db.NoAutoUpd
	a.db.mu.RUnlock()
	return map[string]any{"ok": true, "central": !off, "auto_update": !noAuto, "version": Version, "update": a.upd.view(), "update_configured": UpdateRepo != "" && UpdatePubKey != ""}
}

func (a *App) adminSettings(w http.ResponseWriter, r *http.Request) {
	if a.needAdmin(w, r) == nil {
		return
	}
	writeJSON(w, 200, a.settingsView())
}

// {central: bool, auto_update: bool}  — central=false یعنی قطع از شبکه‌ی مرکزی: کپچا خاموش + بدون به‌روزرسانی
func (a *App) adminSettingsSet(w http.ResponseWriter, r *http.Request) {
	if a.needAdmin(w, r) == nil {
		return
	}
	var b struct {
		Central    *bool `json:"central"`
		AutoUpdate *bool `json:"auto_update"`
	}
	if !decodeJSON(w, r, &b) {
		return
	}
	a.db.mu.Lock()
	if b.Central != nil {
		a.db.CentralOff = !*b.Central
	}
	if b.AutoUpdate != nil {
		a.db.NoAutoUpd = !*b.AutoUpdate
	}
	a.db.touch()
	a.db.mu.Unlock()
	writeJSON(w, 200, a.settingsView())
}

func (a *App) adminUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if a.needAdmin(w, r) == nil {
		return
	}
	if !a.central() {
		fail(w, 409, "اتصال به شبکه‌ی مرکزی آوند قطع است؛ ابتدا آن را روشن کنید.")
		return
	}
	go a.checkUpdate(true)
	writeJSON(w, 200, map[string]any{"ok": true})
}

// خروج از برنامه (فقط مدیر و فقط از همین رایانه)
func (a *App) adminQuit(w http.ResponseWriter, r *http.Request) {
	if a.needAdmin(w, r) == nil {
		return
	}
	if !isLoopback(r) {
		fail(w, 403, "خروج از آوند فقط از روی همان رایانه‌ای که آوند روی آن اجراست ممکن است.")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
	select {
	case quitCh <- struct{}{}:
	default:
	}
}
