//go:build windows

package autostart

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	registryKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName       = "Aether"
)

func exePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}

// IsEnabled checks whether the Aether auto-start registry entry exists.
func IsEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(valueName)
	return err == nil
}

// Enable creates the auto-start registry entry pointing to the current executable.
func Enable() error {
	exe, err := exePath()
	if err != nil {
		return err
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, registryKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.SetStringValue(valueName, exe)
}

// Disable removes the auto-start registry entry.
func Disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.DeleteValue(valueName)
}

// Toggle switches the auto-start state and returns the new state.
func Toggle() (enabled bool, err error) {
	if IsEnabled() {
		return false, Disable()
	}
	return true, Enable()
}
