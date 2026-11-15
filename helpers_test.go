package main

import (
	"bytes"
	"image"
	"io"
	"net/http"
	"os"
	"runtime"
	"testing"
)

func mkdir(p string) error { return os.MkdirAll(p, 0o700) }
func goos() string         { return runtime.GOOS }
func goarch() string       { return runtime.GOARCH }
func exe() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
func mustRead(r *http.Response) []byte { b, _ := io.ReadAll(r.Body); return b }
func decodeCfg(t *testing.T, p string) image.Config {
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	c, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
