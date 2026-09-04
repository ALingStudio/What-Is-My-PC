//go:build windows

// Package psexec 创建隐藏的 PowerShell 进程（不弹出控制台窗口）。
package psexec

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000 // CREATE_NO_WINDOW

// Command 返回一个完全静默（无窗口）的 powershell 进程。
func Command(args ...string) *exec.Cmd {
	cmd := exec.Command("powershell", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNoWindow,
	}
	return cmd
}
