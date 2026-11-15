package main

import (
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var thumbSem = make(chan struct{}, 2)
var thumbMu sync.Mutex

const thumbMax = 360

// جهت عکس از EXIF (گوشی‌ها عکس را چرخیده ذخیره می‌کنند)
func exifOrientation(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 1
	}
	defer f.Close()
	buf := make([]byte, 128<<10)
	n, _ := f.Read(buf)
	b := buf[:n]
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 < len(b) {
		if b[i] != 0xFF {
			return 1
		}
		m, l := b[i+1], int(binary.BigEndian.Uint16(b[i+2:]))
		if m == 0xE1 && i+10 < len(b) && string(b[i+4:i+8]) == "Exif" {
			t := b[i+10:]
			end := i + 2 + l
			if end < len(b) {
				t = b[i+10 : end]
			}
			if len(t) < 8 {
				return 1
			}
			var bo binary.ByteOrder = binary.LittleEndian
			if t[0] == 'M' {
				bo = binary.BigEndian
			}
			off := int(bo.Uint32(t[4:]))
			if off+2 > len(t) {
				return 1
			}
			cnt := int(bo.Uint16(t[off:]))
			for k := 0; k < cnt; k++ {
				e := off + 2 + k*12
				if e+12 > len(t) {
					return 1
				}
				if bo.Uint16(t[e:]) == 0x0112 {
					if v := int(bo.Uint16(t[e+8:])); v >= 1 && v <= 8 {
						return v
					}
				}
			}
			return 1
		}
		if m == 0xDA {
			return 1
		}
		i += 2 + l
	}
	return 1
}

func downscale(src image.Image, max int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if w > h && w > max {
		dw, dh = max, h*max/w
	} else if h >= w && h > max {
		dw, dh = w*max/h, max
	}
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	sx, sy := float64(w)/float64(dw), float64(h)/float64(dh)
	ss := int(sx)
	if ss < 1 {
		ss = 1
	}
	if ss > 4 {
		ss = 4
	}
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var r, g, bl, al, cnt uint32
			for j := 0; j < ss; j++ {
				for i := 0; i < ss; i++ {
					px := b.Min.X + int((float64(x)+(float64(i)+0.5)/float64(ss))*sx)
					py := b.Min.Y + int((float64(y)+(float64(j)+0.5)/float64(ss))*sy)
					cr, cg, cb, ca := src.At(px, py).RGBA()
					r, g, bl, al, cnt = r+cr, g+cg, bl+cb, al+ca, cnt+1
				}
			}
			dst.SetRGBA(x, y, color.RGBA{uint8(r / cnt >> 8), uint8(g / cnt >> 8), uint8(bl / cnt >> 8), uint8(al / cnt >> 8)})
		}
	}
	return dst
}

func orient(src *image.RGBA, o int) *image.RGBA {
	if o <= 1 {
		return src
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2:
				nx, ny = w-1-x, y
			case 3:
				nx, ny = w-1-x, h-1-y
			case 4:
				nx, ny = x, h-1-y
			case 5:
				nx, ny = y, x
			case 6:
				nx, ny = h-1-y, x
			case 7:
				nx, ny = h-1-y, w-1-x
			case 8:
				nx, ny = y, w-1-x
			}
			dst.Set(nx, ny, src.At(x, y))
		}
	}
	return dst
}

func (a *App) serveThumb(w http.ResponseWriter, r *http.Request) {
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
	if f == nil || cp.Owner != u.ID || cp.Kind != "image" {
		http.NotFound(w, r)
		return
	}
	if m := cp.Mime; m != "image/jpeg" && m != "image/png" && m != "image/gif" {
		http.NotFound(w, r) // مرورگر خودش نمایش می‌دهد (webp/svg/…)
		return
	}
	cache := filepath.Join(a.dir, "thumbs", cp.ID+".jpg")
	if _, err := os.Stat(cache); err != nil {
		thumbSem <- struct{}{}
		err := makeThumb(filepath.Join(a.dir, "files", cp.ID), cache, cp.Mime)
		<-thumbSem
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, cache)
}

func makeThumb(src, dst, mt string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil || cfg.Width*cfg.Height > 120_000_000 {
		return os.ErrInvalid
	}
	f.Seek(0, 0)
	var img image.Image
	switch {
	case strings.HasSuffix(mt, "jpeg"):
		img, err = jpeg.Decode(f)
	case strings.HasSuffix(mt, "png"):
		img, err = png.Decode(f)
	default:
		img, err = gif.Decode(f)
	}
	if err != nil {
		return err
	}
	o := 1
	if strings.HasSuffix(mt, "jpeg") {
		o = exifOrientation(src)
	}
	out := orient(downscale(img, thumbMax), o)
	tmp := dst + ".tmp"
	t, err := os.Create(tmp)
	if err != nil {
		return err
	}
	// پس‌زمینه‌ی سفید برای PNG شفاف
	bg := image.NewRGBA(out.Bounds())
	for i := range bg.Pix {
		bg.Pix[i] = 255
	}
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			c := out.RGBAAt(x, y)
			inv := 255 - uint32(c.A) // RGBA پیش‌ضرب‌شده است: روی سفید = c + (255-a)
			bg.SetRGBA(x, y, color.RGBA{uint8(uint32(c.R) + inv), uint8(uint32(c.G) + inv), uint8(uint32(c.B) + inv), 255})
		}
	}
	err = jpeg.Encode(t, bg, &jpeg.Options{Quality: 80})
	t.Close()
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
