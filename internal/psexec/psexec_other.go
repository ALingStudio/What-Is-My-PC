//go:build !windows

// Package psexec 非 Windows 平台的占位实现（本项目只构建 Windows 目标）。
package psexec

import "os/exec"

// Command 普通进程（非 Windows 平台无控制台窗口问题）。
func Command(args ...string) *exec.Cmd {
	return exec.Command("powershell", args...)
}
