package provider

import (
	"os"
	"path/filepath"
	"strings"
)

func defaultCredentialsDir(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return name
	}
	return filepath.Join(home, name)
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func accountName(providerName, credentialsDir, defaultDirName string) string {
	if filepath.Clean(credentialsDir) == filepath.Clean(defaultCredentialsDir(defaultDirName)) {
		return providerName
	}
	label := filepath.Base(filepath.Clean(credentialsDir))
	return providerName + " (" + label + ")"
}
