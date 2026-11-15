package main

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var extKind = map[string]string{}

func init() {
	add := func(kind string, exts ...string) {
		for _, e := range exts {
			extKind[e] = kind
		}
	}
	add("image", "jpg", "jpeg", "png", "gif", "webp", "bmp", "avif", "heic", "heif", "svg", "tif", "tiff", "jfif")
	add("video", "mp4", "mkv", "webm", "mov", "avi", "m4v", "wmv", "flv", "ts", "3gp", "mpg", "mpeg", "ogv")
	add("audio", "mp3", "wav", "flac", "aac", "m4a", "ogg", "opus", "wma")
	add("doc", "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "odt", "ods", "odp", "txt", "rtf", "csv", "md", "json", "xml", "html", "htm", "log")
	for e, m := range map[string]string{
		".mp4": "video/mp4", ".m4v": "video/mp4", ".webm": "video/webm", ".mkv": "video/x-matroska", ".mov": "video/quicktime",
		".avi": "video/x-msvideo", ".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".flac": "audio/flac", ".ogg": "audio/ogg", ".opus": "audio/ogg",
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp", ".svg": "image/svg+xml",
		".heic": "image/heic", ".avif": "image/avif", ".pdf": "application/pdf", ".txt": "text/plain; charset=utf-8", ".md": "text/plain; charset=utf-8",
		".csv": "text/plain; charset=utf-8", ".json": "text/plain; charset=utf-8", ".xml": "text/plain; charset=utf-8", ".log": "text/plain; charset=utf-8",
	} {
		_ = mime.AddExtensionType(e, m)
	}
}

func kindOf(name string) (kind, mt string) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	kind = extKind[ext]
	if kind == "" {
		kind = "other"
	}
	mt = mime.TypeByExtension("." + ext)
	if mt == "" {
		mt = "application/octet-stream"
	}
	return
}

func cleanName(s string) string {
	s = strings.ReplaceAll(s, "\\", "/")
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"|?*`, r) {
			continue
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(strings.Trim(b.String(), "."))
	if r := []rune(out); len(r) > 180 {
		out = string(r[:180])
	}
	return out
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v) != nil {
		fail(w, 400, "درخواست نامعتبر")
		return false
	}
	return true
}

func (a *App) fileRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/files", a.listFiles)
	mux.HandleFunc("POST /api/upload", a.upload)
	mux.HandleFunc("POST /api/files/delete", a.deleteFiles)
	mux.HandleFunc("POST /api/files/move", a.moveFiles)
	mux.HandleFunc("POST /api/files/rename", a.renameFile)
	mux.HandleFunc("GET /api/folders", a.listFolders)
	mux.HandleFunc("POST /api/folders", a.saveFolder)
	mux.HandleFunc("POST /api/folders/delete", a.deleteFolder)
	mux.HandleFunc("GET /f/{id}", a.serveFile)
	mux.HandleFunc("GET /t/{id}", a.serveThumb)
}

func (a *App) listFiles(w http.ResponseWriter, r *http.Request) {
	u := a.needUser(w, r)
	if u == nil {
		return
	}
	q := r.URL.Query()
	kind, folder, search, sortBy := q.Get("kind"), q.Get("folder"), strings.ToLower(strings.TrimSpace(q.Get("q"))), q.Get("sort")
	a.db.mu.RLock()
	all := a.db.sortedFiles(u.ID)
	a.db.mu.RUnlock()
	out := make([]*File, 0, len(all))
	for _, f := range all {
		if kind != "" && f.Kind != kind {
			continue
		}
		if folder == "none" && f.Folder != "" || folder != "" && folder != "none" && f.Folder != folder {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(f.Name), search) {
			continue
		}
		out = append(out, f)
	}
	switch sortBy {
	case "name":
		sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	case "size":
		sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	case "taken":
		sort.Slice(out, func(i, j int) bool { return out[i].Taken > out[j].Taken })
	}
	if len(out) > 20000 {
		out = out[:20000]
	}
	writeJSON(w, 200, map[string]any{"ok": true, "files": out})
}

func (a *App) upload(w http.ResponseWriter, r *http.Request) {
	u := a.needUser(w, r)
	if u == nil {
		return
	}
	q := r.URL.Query()
	name := cleanName(q.Get("name"))
	if name == "" {
		fail(w, 400, "نام فایل نامعتبر است.")
		return
	}
	folder := q.Get("folder")
	a.db.mu.RLock()
	if folder != "" {
		if f := a.db.folderByID(folder); f == nil || f.Owner != u.ID {
			folder = ""
		}
	}
	cur := a.db.userByID(u.ID)
	quota, used := cur.Quota, a.db.usedBytes(u.ID)
	a.db.mu.RUnlock()
	remaining := int64(-1)
	if quota > 0 {
		remaining = quota - used
		if remaining <= 0 || (r.ContentLength > 0 && r.ContentLength > remaining) {
			fail(w, 413, "فضای ذخیره‌سازی شما کافی نیست.")
			return
		}
	}
	a.uploads.Add(1)
	defer a.uploads.Add(-1)
	tmp, err := os.CreateTemp(filepath.Join(a.dir, "tmp"), "up-*")
	if err != nil {
		fail(w, 500, "خطا در ذخیره‌ی فایل.")
		return
	}
	var body io.Reader = r.Body
	if remaining >= 0 {
		body = http.MaxBytesReader(w, r.Body, remaining)
	}
	n, err := io.Copy(tmp, body)
	tmp.Close()
	if err != nil {
		os.Remove(tmp.Name())
		if remaining >= 0 && n >= remaining {
			fail(w, 413, "فضای ذخیره‌سازی شما کافی نیست.")
		} else {
			fail(w, 400, "آپلود ناقص ماند.")
		}
		return
	}
	id := randHex(8)
	kind, mt := kindOf(name)
	taken, _ := strconv.ParseInt(q.Get("mtime"), 10, 64)
	if taken <= 0 || taken > now()+86400 {
		taken = now()
	}
	a.db.mu.Lock()
	if quota > 0 && a.db.usedBytes(u.ID)+n > quota { // رقابت چند آپلود همزمان
		a.db.mu.Unlock()
		os.Remove(tmp.Name())
		fail(w, 413, "فضای ذخیره‌سازی شما کافی نیست.")
		return
	}
	if err := os.Rename(tmp.Name(), filepath.Join(a.dir, "files", id)); err != nil {
		a.db.mu.Unlock()
		os.Remove(tmp.Name())
		fail(w, 500, "خطا در ذخیره‌ی فایل.")
		return
	}
	f := &File{ID: id, Owner: u.ID, Name: name, Mime: mt, Kind: kind, Size: n, Created: now(), Taken: taken, Folder: folder}
	a.db.Files = append(a.db.Files, f)
	a.db.touch()
	a.db.mu.Unlock()
	writeJSON(w, 200, map[string]any{"ok": true, "file": f})
}

func (a *App) removeFileData(id string) {
	_ = os.Remove(filepath.Join(a.dir, "files", id))
	_ = os.Remove(filepath.Join(a.dir, "thumbs", id+".jpg"))
}

func (a *App) deleteFiles(w http.ResponseWriter, r *http.Request) {
	u := a.needUser(w, r)
	if u == nil {
		return
	}
	var b struct {
		IDs []string `json:"ids"`
	}
	if !decodeJSON(w, r, &b) {
		return
	}
	set := map[string]bool{}
	for _, id := range b.IDs {
		set[id] = true
	}
	var gone []string
	a.db.mu.Lock()
	keep := a.db.Files[:0]
	for _, f := range a.db.Files {
		if set[f.ID] && f.Owner == u.ID {
			gone = append(gone, f.ID)
			continue
		}
		keep = append(keep, f)
	}
	a.db.Files = keep
	a.db.touch()
	a.db.mu.Unlock()
	for _, id := range gone {
		a.removeFileData(id)
	}
	writeJSON(w, 200, map[string]any{"ok": true, "deleted": len(gone)})
}

func (a *App) moveFiles(w http.ResponseWriter, r *http.Request) {
	u := a.needUser(w, r)
	if u == nil {
		return
	}
	var b struct {
		IDs    []string `json:"ids"`
		Folder string   `json:"folder"`
	}
	if !decodeJSON(w, r, &b) {
		return
	}
	set := map[string]bool{}
	for _, id := range b.IDs {
		set[id] = true
	}
	a.db.mu.Lock()
	defer a.db.mu.Unlock()
	if b.Folder != "" {
		if f := a.db.folderByID(b.Folder); f == nil || f.Owner != u.ID {
			fail(w, 404, "پوشه پیدا نشد.")
			return
		}
	}
	for _, f := range a.db.Files {
		if set[f.ID] && f.Owner == u.ID {
			f.Folder = b.Folder
		}
	}
	a.db.touch()
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) renameFile(w http.ResponseWriter, r *http.Request) {
	u := a.needUser(w, r)
	if u == nil {
		return
	}
	var b struct{ ID, Name string }
	if !decodeJSON(w, r, &b) {
		return
	}
	name := cleanName(b.Name)
	if name == "" {
		fail(w, 400, "نام نامعتبر است.")
		return
	}
	a.db.mu.Lock()
	defer a.db.mu.Unlock()
	f := a.db.fileByID(b.ID)
	if f == nil || f.Owner != u.ID {
		fail(w, 404, "فایل پیدا نشد.")
		return
	}
	f.Name = name
	f.Kind, f.Mime = kindOf(name)
	a.db.touch()
	writeJSON(w, 200, map[string]any{"ok": true, "file": f})
}

func (a *App) listFolders(w http.ResponseWriter, r *http.Request) {
	u := a.needUser(w, r)
	if u == nil {
		return
	}
	type fo struct {
		*Folder
		Count int `json:"count"`
	}
	out := []fo{}
	a.db.mu.RLock()
	for _, f := range a.db.Folders {
		if f.Owner == u.ID {
			c := 0
			for _, x := range a.db.Files {
				if x.Owner == u.ID && x.Folder == f.ID {
					c++
				}
			}
			out = append(out, fo{f, c})
		}
	}
	a.db.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	writeJSON(w, 200, map[string]any{"ok": true, "folders": out})
}

func (a *App) saveFolder(w http.ResponseWriter, r *http.Request) {
	u := a.needUser(w, r)
	if u == nil {
		return
	}
	var b struct{ ID, Name string }
	if !decodeJSON(w, r, &b) {
		return
	}
	name := cleanName(b.Name)
	if name == "" || len([]rune(name)) > 60 {
		fail(w, 400, "نام پوشه نامعتبر است.")
		return
	}
	a.db.mu.Lock()
	defer a.db.mu.Unlock()
	for _, f := range a.db.Folders {
		if f.Owner == u.ID && strings.EqualFold(f.Name, name) && f.ID != b.ID {
			fail(w, 409, "پوشه‌ای با این نام دارید.")
			return
		}
	}
	if b.ID != "" {
		f := a.db.folderByID(b.ID)
		if f == nil || f.Owner != u.ID {
			fail(w, 404, "پوشه پیدا نشد.")
			return
		}
		f.Name = name
	} else {
		a.db.Folders = append(a.db.Folders, &Folder{ID: randHex(6), Owner: u.ID, Name: name, Created: now()})
	}
	a.db.touch()
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) deleteFolder(w http.ResponseWriter, r *http.Request) {
	u := a.needUser(w, r)
	if u == nil {
		return
	}
	var b struct{ ID string }
	if !decodeJSON(w, r, &b) {
		return
	}
	a.db.mu.Lock()
	defer a.db.mu.Unlock()
	keep := a.db.Folders[:0]
	for _, f := range a.db.Folders {
		if f.ID == b.ID && f.Owner == u.ID {
			continue // فایل‌ها حذف نمی‌شوند؛ فقط از پوشه بیرون می‌آیند
		}
		keep = append(keep, f)
	}
	a.db.Folders = keep
	for _, f := range a.db.Files {
		if f.Owner == u.ID && f.Folder == b.ID {
			f.Folder = ""
		}
	}
	a.db.touch()
	writeJSON(w, 200, map[string]any{"ok": true})
}

// دانلود / پخش (با Range برای ویدیو)؛ فقط صاحب فایل
func (a *App) serveFile(w http.ResponseWriter, r *http.Request) {
	u := a.currentUser(r)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	a.db.mu.RLock()
	f := a.db.fileByID(r.PathValue("id"))
	var cp File
	if f != nil {
		cp = *f
	}
	a.db.mu.RUnlock()
	if f == nil || cp.Owner != u.ID {
		http.NotFound(w, r)
		return
	}
	fh, err := os.Open(filepath.Join(a.dir, "files", cp.ID))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer fh.Close()
	h := w.Header()
	h.Set("Content-Type", cp.Mime)
	h.Set("Cache-Control", "private, max-age=3600")
	inlineOK := strings.HasPrefix(cp.Mime, "image/") || strings.HasPrefix(cp.Mime, "video/") || strings.HasPrefix(cp.Mime, "audio/") ||
		cp.Mime == "application/pdf" || strings.HasPrefix(cp.Mime, "text/plain")
	disp := "attachment"
	if inlineOK && r.URL.Query().Get("dl") != "1" {
		disp = "inline"
	}
	h.Set("Content-Disposition", disp+"; filename*=UTF-8''"+strings.ReplaceAll(url.PathEscape(cp.Name), "+", "%20"))
	if cp.Mime != "application/pdf" {
		h.Set("Content-Security-Policy", "sandbox; default-src 'none'; style-src 'unsafe-inline'")
	}
	http.ServeContent(w, r, cp.Name, time.Unix(cp.Created, 0), fh)
}
