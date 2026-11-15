package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type User struct {
	ID      string `json:"id"`
	Name    string `json:"username"`
	Hash    string `json:"hash"`
	Salt    string `json:"salt"`
	Iter    int    `json:"iter"`
	Admin   bool   `json:"admin"`
	Active  bool   `json:"active"`
	Quota   int64  `json:"quota"` // بایت؛ ۰ = نامحدود
	Created int64  `json:"created"`
}

type Code struct {
	Code    string `json:"code"`
	Note    string `json:"note"`
	Quota   int64  `json:"quota"`
	Admin   bool   `json:"admin"`
	UsedBy  string `json:"used_by"`
	UsedAt  int64  `json:"used_at"`
	Created int64  `json:"created"`
}

type Session struct {
	Hash    string `json:"hash"`
	UserID  string `json:"user_id"`
	Expires int64  `json:"expires"`
}

type File struct {
	ID      string `json:"id"`
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Mime    string `json:"mime"`
	Kind    string `json:"kind"` // image | video | audio | doc | other
	Size    int64  `json:"size"`
	Created int64  `json:"created"`
	Taken   int64  `json:"taken"` // زمان تغییر فایل روی دستگاه فرستنده (برای مرتب‌سازی عکس‌ها)
	Folder  string `json:"folder"`
}

type Folder struct {
	ID      string `json:"id"`
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Created int64  `json:"created"`
}

type DB struct {
	mu      sync.RWMutex
	path    string
	timer   *time.Timer
	Users   []*User    `json:"users"`
	Codes   []*Code    `json:"codes"`
	Session []*Session `json:"sessions"`
	Files   []*File    `json:"files"`
	Folders []*Folder  `json:"folders"`
	// تنظیمات سراسری (فقط مدیر)
	CentralOff bool `json:"central_off"` // true = قطع از «شبکه‌ی مرکزی آوند»: کپچا خاموش، بدون به‌روزرسانی خودکار
	NoAutoUpd  bool `json:"no_auto_update"`
}

func openDB(dir string) (*DB, error) {
	d := &DB{path: filepath.Join(dir, "db.json")}
	b, err := os.ReadFile(d.path)
	if err == nil {
		if err := json.Unmarshal(b, d); err != nil {
			// فایل خراب: نسخه‌ی پشتیبان بگیر و خالی شروع کن (داده‌ی کاربران پاک نشود)
			_ = os.Rename(d.path, d.path+".corrupt-"+time.Now().Format("20060102-150405"))
			d = &DB{path: d.path}
		}
	}
	return d, nil
}

// ذخیره‌ی با تأخیر کوتاه؛ نوشتن اتمیک (فایل موقت + rename). باید با قفل نوشتن صدا زده شود.
func (d *DB) touch() {
	if d.timer != nil {
		return
	}
	d.timer = time.AfterFunc(300*time.Millisecond, func() {
		d.mu.Lock()
		d.timer = nil
		d.mu.Unlock()
		d.Flush()
	})
}

func (d *DB) Flush() {
	d.mu.RLock()
	b, err := json.Marshal(d)
	d.mu.RUnlock()
	if err != nil {
		return
	}
	tmp := d.path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, d.path)
	}
}

// ---- کمکی‌های جستجو (قفل را صدا زننده می‌گیرد) ----
func (d *DB) userByID(id string) *User {
	for _, u := range d.Users {
		if u.ID == id {
			return u
		}
	}
	return nil
}

func (d *DB) userByName(name string) *User {
	for _, u := range d.Users {
		if equalFold(u.Name, name) {
			return u
		}
	}
	return nil
}

func (d *DB) fileByID(id string) *File {
	for _, f := range d.Files {
		if f.ID == id {
			return f
		}
	}
	return nil
}

func (d *DB) folderByID(id string) *Folder {
	for _, f := range d.Folders {
		if f.ID == id {
			return f
		}
	}
	return nil
}

func (d *DB) usedBytes(owner string) int64 {
	var n int64
	for _, f := range d.Files {
		if f.Owner == owner {
			n += f.Size
		}
	}
	return n
}

func (d *DB) sortedFiles(owner string) []*File {
	out := []*File{}
	for _, f := range d.Files {
		if f.Owner == owner {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	return out
}
