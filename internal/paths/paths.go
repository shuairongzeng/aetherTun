package paths

import (
	"os"
	"path/filepath"
)

type AppPaths struct {
	RootDir    string
	ConfigFile string
	LogDir     string
	RuntimeDir string
}

func DefaultPaths() AppPaths {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = "."
	}

	rootDir := filepath.Join(localAppData, "Aether")
	return AppPaths{
		RootDir:    rootDir,
		ConfigFile: filepath.Join(rootDir, "config.json"),
		LogDir:     filepath.Join(rootDir, "logs"),
		RuntimeDir: filepath.Join(rootDir, "run"),
	}
}

func EnsureAppDirs(appPaths AppPaths) error {
	dirs := []string{
		appPaths.RootDir,
		appPaths.LogDir,
		appPaths.RuntimeDir,
		filepath.Dir(appPaths.ConfigFile),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}
