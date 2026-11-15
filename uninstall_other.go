//go:build !windows

package main

import "fmt"

func uninstallSelf(silent bool) { fmt.Println("uninstall is only available on Windows") }
