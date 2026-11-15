//go:build windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func hidden(c *exec.Cmd) *exec.Cmd {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return c
}

// avand uninstall [/S]  — حذف برنامه؛ پوشه‌ی داده (فایل‌های کاربران) هرگز حذف نمی‌شود
func uninstallSelf(silent bool) {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	if !silent {
		fmt.Println("Avand will be removed from:", dir)
		fmt.Println("Your files (the data folder) are NOT deleted.")
		fmt.Print("Continue? [y/N] ")
		l, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(l)), "y") {
			return
		}
	}
	_ = hidden(exec.Command("taskkill", "/F", "/FI", "IMAGENAME eq avand.exe", "/FI", "PID ne "+fmt.Sprint(os.Getpid()))).Run()
	ps := `$d=[Environment]::GetFolderPath('Desktop'),(Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\Avand'); ` +
		`Remove-Item -Force -ErrorAction SilentlyContinue (Join-Path $d[0] 'Avand.lnk'); Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $d[1]`
	_ = hidden(exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps)).Run()
	_ = hidden(exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Avand`, "/f")).Run()
	_ = hidden(exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "Avand", "/f")).Run()
	_ = hidden(exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name=Avand")).Run()
	// خودحذفی بعد از خروج این پروسه؛ فقط فایل‌های خود برنامه، نه پوشه‌ی داده
	cmd := fmt.Sprintf(`ping -n 3 127.0.0.1 >nul & del /f /q "%s" "%s" & rmdir "%s"`, exe, filepath.Join(dir, "avand.path"), dir)
	_ = hidden(exec.Command("cmd", "/C", cmd)).Start()
	if !silent {
		fmt.Println("Done.")
	}
}
