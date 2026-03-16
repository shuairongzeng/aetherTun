//go:build windows

package launcher

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var shellExecuteW = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")

type LaunchOptions struct {
	ConfigPath  string
	ControlPort int
	Token       string
}

type launchSpec struct {
	Verb       string
	File       string
	Parameters string
	Directory  string
	ShowCmd    int
}

func buildLaunchSpec(corePath string, options LaunchOptions) launchSpec {
	return launchSpec{
		Verb:       "runas",
		File:       corePath,
		Parameters: fmt.Sprintf(`--config %s --control-port %d --token %s`, quoteArg(options.ConfigPath), options.ControlPort, options.Token),
		Directory:  filepath.Dir(corePath),
		ShowCmd:    0,
	}
}

func LaunchElevatedCore(corePath string, options LaunchOptions) error {
	spec := buildLaunchSpec(corePath, options)

	result, _, callErr := shellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(spec.Verb))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(spec.File))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(spec.Parameters))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(spec.Directory))),
		uintptr(spec.ShowCmd),
	)
	if result <= 32 {
		if callErr != syscall.Errno(0) {
			return fmt.Errorf("ShellExecuteW failed: %w", callErr)
		}
		return fmt.Errorf("ShellExecuteW failed with code %d", result)
	}

	return nil
}

func quoteArg(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}
