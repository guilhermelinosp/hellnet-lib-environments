package environments

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetString(t *testing.T) {
	t.Setenv("MY_PREFIX_HELLNET_KEY", "primary")
	t.Setenv("MY_FALLBACK_HELLNET_KEY", "fallback")

	if got := GetString("MY_PREFIX_", "MY_FALLBACK_", "KEY", "def"); got != "primary" {
		t.Fatalf("expected primary, got %q", got)
	}

	os.Unsetenv("MY_PREFIX_HELLNET_KEY")
	if got := GetString("MY_PREFIX_", "MY_FALLBACK_", "KEY", "def"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}

	os.Unsetenv("MY_FALLBACK_HELLNET_KEY")
	if got := GetString("MY_PREFIX_", "MY_FALLBACK_", "KEY", "def"); got != "def" {
		t.Fatalf("expected def, got %q", got)
	}

	// No fallback prefix.
	if got := GetString("MISSING_", "", "KEY", "onlydef"); got != "onlydef" {
		t.Fatalf("expected onlydef, got %q", got)
	}

	// Empty primary prefix reads the bare suffix variable.
	t.Setenv("HELLNET_BARE_KEY", "bare")
	if got := GetString("", "MY_FALLBACK_", "BARE_KEY", "def"); got != "bare" {
		t.Fatalf("expected bare, got %q", got)
	}
}

func TestGetInt(t *testing.T) {
	t.Setenv("INT_KEY", "42")
	if got := GetInt("INT_KEY", "7"); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}

	os.Unsetenv("INT_KEY")
	if got := GetInt("INT_KEY", "7"); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}

	// Missing with no default panics.
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic when required var missing")
			}
		}()
		GetInt("NEVER_SET_KEY")
	}()

	// Non-integer value panics.
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for invalid integer")
			}
		}()
		t.Setenv("INT_KEY", "notanint")
		GetInt("INT_KEY", "7")
	}()
}

func TestGetBool(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"on", true},
		{"TRUE", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
	}
	for _, c := range cases {
		t.Setenv("B_KEY", c.val)
		if got := GetBool("B_KEY", "false"); got != c.want {
			t.Fatalf("value %q: expected %v, got %v", c.val, c.want, got)
		}
	}

	os.Unsetenv("B_KEY")
	if got := GetBool("B_KEY", "true"); got != true {
		t.Fatalf("expected default true, got %v", got)
	}
}

func TestGetDurationGoFormat(t *testing.T) {
	// Go duration format is supported by GetDuration.
	t.Setenv("DUR_KEY", "1h30m")
	if got := GetDuration("DUR_KEY", "1s"); got != 90*time.Minute {
		t.Fatalf("expected 90m, got %v", got)
	}
}

func TestGetDuration(t *testing.T) {
	t.Setenv("DUR_KEY", "1h30m")
	if got := GetDuration("DUR_KEY", "1s"); got != 90*time.Minute {
		t.Fatalf("expected 90m, got %v", got)
	}

	t.Setenv("DUR_KEY", "5.5s")
	if got := GetDuration("DUR_KEY", "1s"); got != 5500*time.Millisecond {
		t.Fatalf("expected 5.5s, got %v", got)
	}

	os.Unsetenv("DUR_KEY")
	if got := GetDuration("DUR_KEY", "1s"); got != time.Second {
		t.Fatalf("expected default 1s, got %v", got)
	}
}

func TestDeploymentEnvAndIsDev(t *testing.T) {
	os.Unsetenv("HELLNET_ENVIRONMENT")
	if DeploymentEnv() != "" {
		t.Fatalf("expected empty DeploymentEnv")
	}
	if !IsDev() {
		t.Fatalf("expected IsDev true when HELLNET_ENVIRONMENT unset")
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Production")
	if DeploymentEnv() != "Production" {
		t.Fatalf("expected Production")
	}
	if IsDev() {
		t.Fatalf("expected IsDev false when Production")
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Staging")
	if IsDev() {
		t.Fatalf("expected IsDev false when Staging")
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	if !IsDev() {
		t.Fatalf("expected IsDev true when Development")
	}

	// Known names are matched case-insensitively and ignoring surrounding space.
	for _, v := range []string{"development", " Local ", "TEST", "dev"} {
		t.Setenv("HELLNET_ENVIRONMENT", v)
		if !IsDev() {
			t.Fatalf("expected IsDev true for %q", v)
		}
	}

	// Unknown values must not enable development behaviour.
	for _, v := range []string{"production", "PRODUCTION", "prod", " Staging", "Homolog", "typo"} {
		t.Setenv("HELLNET_ENVIRONMENT", v)
		if IsDev() {
			t.Fatalf("expected IsDev false for %q", v)
		}
	}
}

func TestLoadDotEnvSkipsUntrustedFiles(t *testing.T) {
	t.Setenv("HELLNET_ENVIRONMENT", "Development")

	// World-writable .env is ignored.
	dir := t.TempDir()
	writable := filepath.Join(dir, ".env")
	if err := os.WriteFile(writable, []byte("UNTRUSTED=yes\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.Chmod(writable, 0o666); err != nil {
		t.Fatalf("chmod .env: %v", err)
	}
	t.Setenv("CUSTOM_ENV_PATH", writable)
	if err := LoadDotEnv("CUSTOM_ENV_PATH"); err == nil {
		t.Fatalf("expected error for world-writable .env")
	}
	if got := os.Getenv("UNTRUSTED"); got != "" {
		t.Fatalf("expected world-writable .env not to be loaded, got UNTRUSTED=%q", got)
	}

	// .env inside a world-writable directory is ignored.
	shared := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(shared, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(shared, 0o777); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	inShared := filepath.Join(shared, ".env")
	if err := os.WriteFile(inShared, []byte("PLANTED=yes\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	t.Setenv("CUSTOM_ENV_PATH", inShared)
	if err := LoadDotEnv("CUSTOM_ENV_PATH"); err == nil {
		t.Fatalf("expected error for .env in world-writable dir")
	}
	if got := os.Getenv("PLANTED"); got != "" {
		t.Fatalf("expected .env in world-writable dir not to be loaded, got PLANTED=%q", got)
	}
}

func TestLoadDotEnvSkipsSymlinkedCandidate(t *testing.T) {
	t.Setenv("HELLNET_ENVIRONMENT", "Development")

	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, ".env"), []byte("BEYOND_SYMLINK=yes\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	child := filepath.Join(base, "child")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// A symlink loop makes os.Stat fail with ELOOP rather than ErrNotExist.
	if err := os.Symlink(".env", filepath.Join(child, ".env")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	t.Chdir(child)

	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("BEYOND_SYMLINK"); got != "yes" {
		t.Fatalf("expected discovery to continue past symlinked .env, got %q", got)
	}
}

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("FOO=bar\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("FOO"); got != "bar" {
		t.Fatalf("expected FOO=bar, got %q", got)
	}
}

func TestLoadDotEnvCustomVarMissingFile(t *testing.T) {
	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Setenv("MY_ENV_FILE", filepath.Join(t.TempDir(), "does-not-exist.env"))

	err := LoadDotEnv("MY_ENV_FILE")
	if err == nil {
		t.Fatalf("expected error when custom var points to a missing file")
	}
	if !strings.Contains(err.Error(), "MY_ENV_FILE") {
		t.Fatalf("error should name the variable, got %v", err)
	}
}

func TestLoadDotEnvNoopInProd(t *testing.T) {
	t.Setenv("HELLNET_ENVIRONMENT", "Production")
	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv prod: %v", err)
	}
}

func TestLoadDotEnvCustomVar(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "custom.env")
	if err := os.WriteFile(envPath, []byte("CUSTOM_VAR_KEY=from-custom\n"), 0o600); err != nil {
		t.Fatalf("write custom env: %v", err)
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Setenv("CUSTOM_ENV_PATH", envPath)
	t.Chdir(dir)

	if err := LoadDotEnv("CUSTOM_ENV_PATH"); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("CUSTOM_VAR_KEY"); got != "from-custom" {
		t.Fatalf("expected from-custom, got %q", got)
	}
}

func TestLoadDotEnvCustomVarSkipsUnsetAndEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("FALLTHROUGH_KEY=from-cwd\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Setenv("EMPTY_ENV_PATH", "")
	t.Chdir(dir)

	if err := LoadDotEnv("UNSET_ENV_PATH", "EMPTY_ENV_PATH"); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("FALLTHROUGH_KEY"); got != "from-cwd" {
		t.Fatalf("expected from-cwd, got %q", got)
	}
}

func TestLoadDotEnvCustomVarLoadError(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	// A directory passes the os.Stat check but cannot be parsed as a .env file.
	t.Setenv("BROKEN_ENV_PATH", dir)
	t.Chdir(dir)

	if err := LoadDotEnv("BROKEN_ENV_PATH"); err == nil {
		t.Fatalf("expected error when custom path is not a readable .env file")
	}
}

func TestLoadDotEnvFromParentDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("PARENT_DIR_KEY=from-parent\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Chdir(sub)

	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("PARENT_DIR_KEY"); got != "from-parent" {
		t.Fatalf("expected from-parent, got %q", got)
	}
}

func TestLoadDotEnvNoFileFound(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HELLNET_ENVIRONMENT", "Development")
	t.Chdir(dir)

	// No .env exists in the temp dir; parents are system dirs without one.
	if err := LoadDotEnv(); err != nil {
		t.Fatalf("expected nil when no .env is found, got %v", err)
	}
}

func TestGetDurationInvalidValueFallsBackToDefault(t *testing.T) {
	// GetDuration panics on unparseable values.
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for invalid duration")
			}
		}()
		t.Setenv("DUR_KEY", "not-a-duration")
		GetDuration("DUR_KEY", "2s")
	}()
}
