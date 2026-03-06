//go:build windows

package main

import (
	_ "embed"
	"os"
	"path/filepath"
)

// wintun.dll 0.14.1 amd64，嵌入二进制，启动时自动释放到 exe 目录
//
//go:embed wintun.dll
var wintunDLL []byte

// extractWintunDLL 将嵌入的 wintun.dll 释放到 exe 同目录。
// 若已存在且大小匹配则跳过，避免重复写入。
func extractWintunDLL() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	dllPath := filepath.Join(filepath.Dir(exePath), "wintun.dll")

	if info, err := os.Stat(dllPath); err == nil && info.Size() == int64(len(wintunDLL)) {
		return nil // 已存在，跳过
	}

	return os.WriteFile(dllPath, wintunDLL, 0644)
}
