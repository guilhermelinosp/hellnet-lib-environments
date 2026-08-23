// Package environments provides a single, standardized way for Hellnet Go
// libraries to read configuration from environment variables and .env files.
//
// It is the shared backend used by hellnet-lib-cache and hellnet-lib-telemetry
// so that env handling is consistent across the ecosystem.
package environments

import (
	"errors"
	"fmt"
	"io/fs"
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
// resolves to a non-empty env var, that value must point to a loadable file,
// which is then loaded; a set-but-unusable path is an error rather than being
// skipped. Otherwise it searches, in order: the .env next to the executable,
// then .env in the working directory and up to maxParentDirs of its parent
// directories, loading the first one found.
//
// A candidate is only loaded when it is trusted: a regular file, not writable
// by other users, in a directory not writable by other users. An untrusted file
// could be planted by another local user to inject configuration (credentials,
// endpoints) into the process; discovered ones are skipped, while an explicitly
// configured customVars path is reported as an error.
func LoadDotEnv(customVars ...string) error {
	if !IsDev() {
		return nil
	}

	if loaded, err := loadFromCustomVars(customVars); loaded || err != nil {
		return err
	}
	return loadFirstFound(envCandidates())
}

// loadFromCustomVars loads the .env pointed to by the first of vars that is set
// to a non-empty value, reporting whether a file was loaded.
func loadFromCustomVars(vars []string) (bool, error) {
	for _, v := range vars {
		p := os.Getenv(v)
		if p == "" {
			continue
		}
		p = filepath.Clean(p)
		if _, err := os.Stat(p); err != nil {
			return false, fmt.Errorf("environments: %s=%q: %w", v, p, err)
		}
		if !isTrustedEnvFile(p) {
			return false, fmt.Errorf("environments: %s=%q: %w", v, p, errUntrustedEnvFile)
		}
		if err := godotenv.Load(p); err != nil {
			return false, fmt.Errorf("environments: load %q: %w", p, err)
		}
		return true, nil
	}
	return false, nil
}

// envCandidates lists the .env paths to try, in order: next to the executable,
// then in the working directory and up to maxParentDirs of its parents.
func envCandidates() []string {
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
	return candidates
}

// loadFirstFound loads the first existing, trusted candidate.
func loadFirstFound(candidates []string) error {
	for _, c := range candidates {
		if _, err := os.Stat(c); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("environments: stat %q: %w", c, err)
		}
		if !isTrustedEnvFile(c) {
			continue
		}
		if err := godotenv.Load(c); err != nil {
			return fmt.Errorf("environments: load %q: %w", c, err)
		}
		return nil
	}
	return nil
}

// errUntrustedEnvFile is returned when an explicitly configured .env path is
// writable by users other than its owner.
var errUntrustedEnvFile = errors.New("refusing to load .env writable by other users")

// isTrustedEnvFile reports whether path is an existing regular file that only
// its owner can modify, inside a directory that only its owner can modify.
// Symlinks are rejected because their target can be swapped after the check.
func isTrustedEnvFile(path string) bool {
	//nolint:gosec // G703: path is a .env candidate, validated here before use
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

// lookup returns the first non-empty value of prefix+suffix or
// fallbackPrefix+suffix, along with the name of the variable it came from.
func lookup(prefix, fallbackPrefix, suffix string) (val, name string, ok bool) {
	name = prefix + suffix
	if v := os.Getenv(name); v != "" {
		return v, name, true
	}
	if fallbackPrefix != "" {
		name = fallbackPrefix + suffix
		if v := os.Getenv(name); v != "" {
			return v, name, true
		}
	}
	return "", "", false
}

// GetString returns the first non-empty of prefix+suffix, fallbackPrefix+suffix,
// or def.
func GetString(prefix, fallbackPrefix, suffix string, def string) string {
	if v, _, ok := lookup(prefix, fallbackPrefix, suffix); ok {
		return v
	}
	return def
}

// GetInt is like GetString but parses an integer with the same precedence.
// Unparseable values fall back to def; use GetIntE to surface parse errors.
func GetInt(prefix, fallbackPrefix string, suffix string, def int) int {
	n, _ := GetIntE(prefix, fallbackPrefix, suffix, def)
	return n
}

// GetIntE is like GetInt but returns an error when the variable is set to a
// value that cannot be parsed as an integer. Unset variables yield (def, nil).
func GetIntE(prefix, fallbackPrefix string, suffix string, def int) (int, error) {
	s, name, ok := lookup(prefix, fallbackPrefix, suffix)
	if !ok {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def, fmt.Errorf("environments: %s: invalid integer %q: %w", name, s, err)
	}
	return n, nil
}

// GetBool is like GetString but parses a boolean with the same precedence.
// Unparseable values fall back to def; use GetBoolE to surface parse errors.
func GetBool(prefix, fallbackPrefix string, suffix string, def bool) bool {
	b, _ := GetBoolE(prefix, fallbackPrefix, suffix, def)
	return b
}

// GetBoolE is like GetBool but returns an error when the variable is set to a
// value that is not a recognized boolean (true/false, 1/0, yes/no, on/off,
// case-insensitive). Unset variables yield (def, nil).
func GetBoolE(prefix, fallbackPrefix string, suffix string, def bool) (bool, error) {
	s, name, ok := lookup(prefix, fallbackPrefix, suffix)
	if !ok {
		return def, nil
	}
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return def, fmt.Errorf("environments: %s: invalid boolean %q", name, s)
	}
}

// GetDuration is like GetString but parses a time.Duration with the same
// precedence. It accepts both Go duration strings and .NET "HH:MM:SS[.FFF]".
// Unparseable values fall back to def; use GetDurationE to surface parse errors.
func GetDuration(prefix, fallbackPrefix string, suffix string, def time.Duration) time.Duration {
	d, _ := GetDurationE(prefix, fallbackPrefix, suffix, def)
	return d
}

// GetDurationE is like GetDuration but returns an error when the variable is
// set to a value that cannot be parsed as a duration. Unset variables yield
// (def, nil).
func GetDurationE(prefix, fallbackPrefix string, suffix string, def time.Duration) (time.Duration, error) {
	s, name, ok := lookup(prefix, fallbackPrefix, suffix)
	if !ok {
		return def, nil
	}
	d, err := ParseDuration(s)
	if err != nil {
		return def, fmt.Errorf("environments: %s: %w", name, err)
	}
	return d, nil
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
