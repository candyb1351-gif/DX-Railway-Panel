// Package config provides configuration management utilities for the DX panel,
// including version information, logging levels, database paths, and environment variable handling.
package config

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

//go:embed version
var version string

//go:embed name
var name string

// LogLevel represents the logging level for the application.
type LogLevel string

// Logging level constants
const (
	Debug   LogLevel = "debug"
	Info    LogLevel = "info"
	Notice  LogLevel = "notice"
	Warning LogLevel = "warning"
	Error   LogLevel = "error"
)

// GetVersion returns the version string of the DX application.
func GetVersion() string {
	return strings.TrimSpace(version)
}

// GetName returns the name of the DX application.
func GetName() string {
	return strings.TrimSpace(name)
}

// GetLogLevel returns the current logging level based on environment variables or defaults to Info.
func GetLogLevel() LogLevel {
	if IsDebug() {
		return Debug
	}
	logLevel := os.Getenv("DX_LOG_LEVEL")
	if logLevel == "" {
		return Info
	}
	return LogLevel(logLevel)
}

// IsDebug returns true if debug mode is enabled via the DX_DEBUG environment variable.
func IsDebug() bool {
	return os.Getenv("DX_DEBUG") == "true"
}

// IsSkipHSTS returns true if skipping HSTS mode is enabled via the DX_SKIP_HSTS environment variable.
func IsSkipHSTS() bool {
	return os.Getenv("DX_SKIP_HSTS") == "true"
}

// GetBinFolderPath returns the path to the binary folder, defaulting to "bin" if not set via DX_BIN_FOLDER.
func GetBinFolderPath() string {
	binFolderPath := os.Getenv("DX_BIN_FOLDER")
	if binFolderPath == "" {
		binFolderPath = "bin"
	}
	return binFolderPath
}

func getBaseDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	exeDir := filepath.Dir(exePath)
	exeDirLower := strings.ToLower(filepath.ToSlash(exeDir))
	if strings.Contains(exeDirLower, "/appdata/local/temp/") || strings.Contains(exeDirLower, "/go-build") {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}
	return exeDir
}

// GetDBFolderPath returns the path to the database folder based on environment variables or platform defaults.
func GetDBFolderPath() string {
	dbFolderPath := os.Getenv("DX_DB_FOLDER")
	if dbFolderPath != "" {
		return dbFolderPath
	}
	if runtime.GOOS == "windows" {
		return getBaseDir()
	}
	return "/etc/DX"
}

// GetDBPath returns the full path to the database file.
func GetDBPath() string {
	return fmt.Sprintf("%s/%s.db", GetDBFolderPath(), GetName())
}

// GetDBKind returns the configured database backend: "sqlite" (default) or "postgres".
func GetDBKind() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DX_DB_TYPE")))
	switch v {
	case "postgres", "postgresql", "pg":
		return "postgres"
	default:
		return "sqlite"
	}
}

// GetDBDSN returns the PostgreSQL DSN from DX_DB_DSN. Empty for sqlite.
func GetDBDSN() string {
	return strings.TrimSpace(os.Getenv("DX_DB_DSN"))
}

// GetEnvFilePaths returns the candidate service environment file paths (the file
// systemd loads via EnvironmentFile) across the supported distro families.
func GetEnvFilePaths() []string {
	if runtime.GOOS == "windows" {
		return nil
	}
	return []string{
		"/etc/default/DX",
		"/etc/conf.d/DX",
		"/etc/sysconfig/DX",
	}
}

// GetLogFolder returns the path to the log folder based on environment variables or platform defaults.
func GetLogFolder() string {
	logFolderPath := os.Getenv("DX_LOG_FOLDER")
	if logFolderPath != "" {
		return logFolderPath
	}
	// Under `go test` the Windows default below is CWD-relative ("./log"), which
	// scatters a log/ directory through the source tree (one per tested package).
	// Redirect test runs to a shared temp folder so the source tree stays clean.
	if testing.Testing() {
		return filepath.Join(os.TempDir(), "DX-test-log")
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(".", "log")
	}
	return "/var/log/DX"
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Sync()
}

func init() {
	if runtime.GOOS != "windows" {
		return
	}
	if os.Getenv("DX_DB_FOLDER") != "" {
		return
	}
	oldDBFolder := "/etc/DX"
	oldDBPath := fmt.Sprintf("%s/%s.db", oldDBFolder, GetName())
	newDBFolder := GetDBFolderPath()
	newDBPath := fmt.Sprintf("%s/%s.db", newDBFolder, GetName())
	_, err := os.Stat(newDBPath)
	if err == nil {
		return // new exists
	}
	_, err = os.Stat(oldDBPath)
	if os.IsNotExist(err) {
		return // old does not exist
	}
	_ = copyFile(oldDBPath, newDBPath) // ignore error
}
