// Package environments provides a single, standardized way for Hellnet Go
// libraries to read configuration from environment variables and .env files.
//
// It is the shared backend used by hellnet-lib-cache and hellnet-lib-telemetry
// so that env handling is consistent across the ecosystem.
package environments

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// maxParentDirs bounds how far LoadDotEnv walks up from the working directory
// looking for a .env file, so discovery cannot reach shared directories such as
// /tmp, /home or the filesystem root.
const maxParentDirs = 8

// devEnvNames are the HELLNET_ENVIRONMENT values recognised as non-production.
// Anything else — including unknown or misspelled values — is treated as
// production so that a typo cannot enable development behaviour.
var devEnvNames = map[string]bool{
	"":            true,
	"development": true,
	"dev":         true,
	"local":       true,
	"test":        true,
	"testing":     true,
}

// DeploymentEnv returns the value of HELLNET_ENVIRONMENT, or "" if unset.
func DeploymentEnv() string {
	return os.Getenv("HELLNET_ENVIRONMENT")
}

// IsDev reports whether the current deployment is a non-production environment.
// It returns true only when HELLNET_ENVIRONMENT is empty or one of the known
// development names ("Development", "Dev", "Local", "Test", "Testing", matched
// case-insensitively). Every other value is treated as production.
func IsDev() bool {
	return devEnvNames[strings.ToLower(strings.TrimSpace(DeploymentEnv()))]
}

// LoadDotEnv loads environment variables from a .env file for local development.
//
// In any non-development environment it is a no-op. When a customVars entry
// resolves to a non-empty env var pointing to an existing file, that file is
// loaded. Otherwise it searches, in order: the .env next to the executable,
// then .env in the working directory and up to maxParentDirs of its parent
// directories, loading the first one found.
//
// Candidates that are not regular files, or that are writable by other users,
// or that live in a directory writable by other users, are skipped: such a file
// could be planted by another local user to inject configuration (credentials,
// endpoints) into the process.
func LoadDotEnv(customVars ...string) error {
	if !IsDev() {
		return nil
	}

	for _, v := range customVars {
		if p := os.Getenv(v); p != "" {
			if p = filepath.Clean(p); isTrustedEnvFile(p) {
				return godotenv.Load(p)
			}
		}
	}

	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), ".env"))
	}
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i <= maxParentDirs; i++ {
			candidates = append(candidates, filepath.Join(dir, ".env"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	for _, c := range candidates {
		if isTrustedEnvFile(c) {
			return godotenv.Load(c)
		}
	}
	return nil
}

// isTrustedEnvFile reports whether path is an existing regular file that only
// its owner can modify, inside a directory that only its owner can modify.
// Symlinks are rejected because their target can be swapped after the check.
func isTrustedEnvFile(path string) bool {
	//nolint:gosec // G703: path is a .env candidate, validated below before use
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() || !ownerOnlyWritable(fi.Mode()) {
		return false
	}
	di, err := os.Stat(filepath.Dir(path))
	return err == nil && di.IsDir() && ownerOnlyWritable(di.Mode())
}

// ownerOnlyWritable reports whether m denies write access to group and others.
func ownerOnlyWritable(m os.FileMode) bool {
	return m.Perm()&0o022 == 0
}

// GetString returns the first non-empty of prefix+suffix, fallbackPrefix+suffix,
// or def.
func GetString(prefix, fallbackPrefix, suffix string, def string) string {
	if v := os.Getenv(prefix + suffix); v != "" {
		return v
	}
	if fallbackPrefix != "" {
		if v := os.Getenv(fallbackPrefix + suffix); v != "" {
			return v
		}
	}
	return def
}

// GetInt is like GetString but parses an integer with the same precedence.
func GetInt(prefix, fallbackPrefix string, suffix string, def int) int {
	s := GetString(prefix, fallbackPrefix, suffix, "")
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

// GetBool is like GetString but parses a boolean with the same precedence.
func GetBool(prefix, fallbackPrefix string, suffix string, def bool) bool {
	s := GetString(prefix, fallbackPrefix, suffix, "")
	if s == "" {
		return def
	}
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return def
	}
}

// GetDuration is like GetString but parses a time.Duration with the same
// precedence. It accepts both Go duration strings and .NET "HH:MM:SS[.FFF]".
func GetDuration(prefix, fallbackPrefix string, suffix string, def time.Duration) time.Duration {
	s := GetString(prefix, fallbackPrefix, suffix, "")
	if s == "" {
		return def
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if d, err := ParseDuration(s); err == nil {
		return d
	}
	return def
}

// ParseDuration parses a duration string trying Go's time.ParseDuration first
// and then the .NET "HH:MM:SS" or "HH:MM:SS.FFF" format.
func ParseDuration(s string) (time.Duration, error) {
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	var h, m, sec int
	var frac float64
	if n, err := fmt.Sscanf(s, "%d:%d:%d%f", &h, &m, &sec, &frac); err == nil && n == 4 {
		return time.Duration(h)*time.Hour +
			time.Duration(m)*time.Minute +
			time.Duration(sec)*time.Second +
			time.Duration(frac*float64(time.Second)), nil
	}
	if n, err := fmt.Sscanf(s, "%d:%d:%d", &h, &m, &sec); err == nil && n == 3 {
		return time.Duration(h)*time.Hour +
			time.Duration(m)*time.Minute +
			time.Duration(sec)*time.Second, nil
	}
	return 0, fmt.Errorf("invalid duration: %q", s)
}
