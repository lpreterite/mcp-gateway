package pool

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		if idx := strings.IndexByte(e, '='); idx > 0 {
			m[e[:idx]] = e[idx+1:]
		}
	}
	return m
}

func TestBuildChildEnv_InheritsParentEnv(t *testing.T) {
	parentEnv := os.Environ()
	result := buildChildEnv(nil)
	resultMap := envSliceToMap(result)

	for _, e := range parentEnv {
		if idx := strings.IndexByte(e, '='); idx > 0 {
			key := e[:idx]
			val := e[idx+1:]
			if got, ok := resultMap[key]; !ok {
				t.Errorf("inherited env missing key %q", key)
			} else if got != val {
				t.Errorf("inherited env key %q: got %q, want %q", key, got, val)
			}
		}
	}
}

func TestBuildChildEnv_MergesExtra(t *testing.T) {
	extra := map[string]string{
		"MCP_CUSTOM_VAR": "custom_value",
	}
	result := buildChildEnv(extra)
	resultMap := envSliceToMap(result)

	if got := resultMap["MCP_CUSTOM_VAR"]; got != "custom_value" {
		t.Errorf("extra var MCP_CUSTOM_VAR: got %q, want %q", got, "custom_value")
	}
}

func TestBuildChildEnv_ExtraOverridesParent(t *testing.T) {
	parentHome := os.Getenv("HOME")
	if parentHome == "" {
		t.Skip("HOME not set in parent environment")
	}

	extra := map[string]string{
		"HOME": "/fake/home/for/test",
	}
	result := buildChildEnv(extra)
	resultMap := envSliceToMap(result)

	if got := resultMap["HOME"]; got != "/fake/home/for/test" {
		t.Errorf("extra should override parent HOME: got %q, want %q", got, "/fake/home/for/test")
	}
}

func TestBuildChildEnv_NilExtra(t *testing.T) {
	result := buildChildEnv(nil)
	if len(result) == 0 {
		t.Error("buildChildEnv(nil) should return non-empty env slice")
	}

	resultMap := envSliceToMap(result)
	if home, ok := resultMap["HOME"]; !ok || home == "" {
		t.Error("buildChildEnv(nil) should ensure HOME is set")
	}
}

func TestBuildChildEnv_EmptyExtra(t *testing.T) {
	result := buildChildEnv(map[string]string{})
	if len(result) == 0 {
		t.Error("buildChildEnv with empty extra should return non-empty env slice")
	}

	resultMap := envSliceToMap(result)
	if home, ok := resultMap["HOME"]; !ok || home == "" {
		t.Error("buildChildEnv with empty extra should ensure HOME is set")
	}
}

func TestBuildChildEnv_ResultFormat(t *testing.T) {
	result := buildChildEnv(map[string]string{
		"TEST_KEY": "test_value",
	})

	for _, e := range result {
		if idx := strings.IndexByte(e, '='); idx <= 0 {
			t.Errorf("env entry has no '=' or empty key: %q", e)
		}
	}
}

func TestEnsureEssentialEnv_HomeFromLoginShell(t *testing.T) {
	envMap := make(map[string]string)
	ensureEssentialEnv(envMap)

	home := envMap["HOME"]
	if home == "" {
		t.Error("HOME should be set when missing from input map")
	}
	if !filepath.IsAbs(home) {
		t.Errorf("HOME should be an absolute path, got %q", home)
	}
}

func TestEnsureEssentialEnv_HomeExistingPreserved(t *testing.T) {
	envMap := map[string]string{
		"HOME": "/custom/home",
	}
	ensureEssentialEnv(envMap)

	if envMap["HOME"] != "/custom/home" {
		t.Errorf("existing HOME should be preserved: got %q, want %q", envMap["HOME"], "/custom/home")
	}
}

func TestEnsureEssentialEnv_UserSet(t *testing.T) {
	envMap := make(map[string]string)
	ensureEssentialEnv(envMap)

	if envMap["USER"] == "" {
		t.Error("USER should be set when missing from input map")
	}
}

func TestEnsureEssentialEnv_PathSet(t *testing.T) {
	envMap := make(map[string]string)
	ensureEssentialEnv(envMap)

	path := envMap["PATH"]
	if path == "" {
		t.Fatal("PATH should be set when missing from input map")
	}
	if !strings.Contains(path, "/usr/bin") {
		t.Errorf("PATH should contain /usr/bin, got %q", path)
	}
	if !strings.Contains(path, "/bin") {
		t.Errorf("PATH should contain /bin, got %q", path)
	}
}

func TestEnsureEssentialEnv_PathExistingPreserved(t *testing.T) {
	envMap := map[string]string{
		"PATH": "/my/custom/path",
	}
	ensureEssentialEnv(envMap)

	if envMap["PATH"] != "/my/custom/path" {
		t.Errorf("existing PATH should be preserved: got %q, want %q", envMap["PATH"], "/my/custom/path")
	}
}

func TestEnsureEssentialEnv_TmpdirSet(t *testing.T) {
	envMap := make(map[string]string)
	ensureEssentialEnv(envMap)

	if envMap["TMPDIR"] == "" {
		t.Error("TMPDIR should be set when missing from input map")
	}
}

func TestEnsureEssentialEnv_TmpdirExistingPreserved(t *testing.T) {
	envMap := map[string]string{
		"TMPDIR": "/custom/tmp",
	}
	ensureEssentialEnv(envMap)

	if envMap["TMPDIR"] != "/custom/tmp" {
		t.Errorf("existing TMPDIR should be preserved: got %q, want %q", envMap["TMPDIR"], "/custom/tmp")
	}
}

func TestEnsureEssentialEnv_LangSet(t *testing.T) {
	envMap := make(map[string]string)
	ensureEssentialEnv(envMap)

	lang := envMap["LANG"]
	if lang == "" {
		t.Error("LANG should be set when missing from input map")
	}
	if !strings.Contains(lang, "UTF-8") && !strings.Contains(lang, "utf8") {
		t.Errorf("LANG should reference UTF-8 encoding, got %q", lang)
	}
}

func TestEnsureEssentialEnv_LangExistingPreserved(t *testing.T) {
	envMap := map[string]string{
		"LANG": "zh_CN.UTF-8",
	}
	ensureEssentialEnv(envMap)

	if envMap["LANG"] != "zh_CN.UTF-8" {
		t.Errorf("existing LANG should be preserved: got %q, want %q", envMap["LANG"], "zh_CN.UTF-8")
	}
}

func TestEnsureEssentialEnv_TermExistingPreserved(t *testing.T) {
	envMap := map[string]string{
		"TERM": "xterm-256color",
	}
	ensureEssentialEnv(envMap)

	if envMap["TERM"] != "xterm-256color" {
		t.Errorf("existing TERM should be preserved: got %q", envMap["TERM"])
	}
}

func TestEnsureEssentialEnv_LoginShellSupplementsMissingVars(t *testing.T) {
	envMap := make(map[string]string)
	ensureEssentialEnv(envMap)

	loginEnv := fetchLoginShellEnv()
	if loginEnv == nil {
		t.Skip("login shell env not available")
	}

	for k, v := range loginEnv {
		if got, ok := envMap[k]; !ok {
			t.Errorf("login shell var %q should be present in result", k)
		} else if got != v {
			t.Errorf("login shell var %q: got %q, want %q", k, got, v)
		}
	}
}

func TestEnsureEssentialEnv_AllEssentialPresent(t *testing.T) {
	envMap := map[string]string{
		"HOME":   "/test/home",
		"USER":   "testuser",
		"PATH":   "/usr/bin:/bin",
		"TMPDIR": "/tmp",
		"LANG":   "en_US.UTF-8",
		"TERM":   "xterm-256color",
	}
	ensureEssentialEnv(envMap)

	if envMap["HOME"] != "/test/home" {
		t.Errorf("HOME should be preserved: got %q", envMap["HOME"])
	}
	if envMap["USER"] != "testuser" {
		t.Errorf("USER should be preserved: got %q", envMap["USER"])
	}
	if envMap["PATH"] != "/usr/bin:/bin" {
		t.Errorf("PATH should be preserved: got %q", envMap["PATH"])
	}
	if envMap["TMPDIR"] != "/tmp" {
		t.Errorf("TMPDIR should be preserved: got %q", envMap["TMPDIR"])
	}
	if envMap["LANG"] != "en_US.UTF-8" {
		t.Errorf("LANG should be preserved: got %q", envMap["LANG"])
	}
	if envMap["TERM"] != "xterm-256color" {
		t.Errorf("TERM should be preserved: got %q", envMap["TERM"])
	}
}

func TestFetchLoginShellEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("login shell env is not typically available on Windows")
	}

	env := fetchLoginShellEnv()
	if env == nil {
		t.Fatal("fetchLoginShellEnv should return non-nil map on this system")
	}

	if len(env) == 0 {
		t.Error("login shell env should contain at least one variable")
	}

	found := false
	for k := range env {
		if k == "HOME" {
			found = true
			break
		}
	}
	if !found {
		t.Error("login shell env should contain HOME")
	}
}

func TestLoginShellEnvCached(t *testing.T) {
	first := make(map[string]string)
	ensureEssentialEnv(first)

	second := make(map[string]string)
	ensureEssentialEnv(second)

	if first["HOME"] != second["HOME"] {
		t.Errorf("cached login shell env should give consistent HOME: %q vs %q", first["HOME"], second["HOME"])
	}
	if first["PATH"] != second["PATH"] {
		t.Errorf("cached login shell env should give consistent PATH: %q vs %q", first["PATH"], second["PATH"])
	}
}

func TestEnsureEssentialEnv_LoginShellDoesNotOverwrite(t *testing.T) {
	envMap := map[string]string{
		"HOME": "/preserved/home",
	}
	ensureEssentialEnv(envMap)

	if envMap["HOME"] != "/preserved/home" {
		t.Errorf("login shell env should not overwrite existing HOME: got %q", envMap["HOME"])
	}
}

func TestEnsureEssentialEnv_LoginShellMissingTmpdirFallsBack(t *testing.T) {
	originalEnv := loginShellEnv

	loginShellEnv = map[string]string{
		"HOME": "/shell/home",
		"USER": "shell-user",
		"PATH": "/shell/bin:/usr/bin",
	}
	loginShellEnvOnce = sync.Once{}
	loginShellEnvOnce.Do(func() {})

	t.Cleanup(func() {
		loginShellEnv = originalEnv
		loginShellEnvOnce = sync.Once{}
	})

	envMap := make(map[string]string)
	ensureEssentialEnv(envMap)

	if envMap["HOME"] != "/shell/home" {
		t.Errorf("HOME should still come from login shell: got %q", envMap["HOME"])
	}
	if envMap["TMPDIR"] == "" {
		t.Error("TMPDIR should fall back when login shell env does not provide it")
	}
	if envMap["LANG"] == "" {
		t.Error("LANG should fall back when login shell env does not provide it")
	}
}

func TestBuildChildEnv_ExtraHighestPriority(t *testing.T) {
	result := buildChildEnv(map[string]string{
		"HOME": "/extra/home",
		"PATH": "/extra/path",
	})
	resultMap := envSliceToMap(result)

	if resultMap["HOME"] != "/extra/home" {
		t.Errorf("extra HOME should have highest priority: got %q", resultMap["HOME"])
	}
	if resultMap["PATH"] != "/extra/path" {
		t.Errorf("extra PATH should have highest priority: got %q", resultMap["PATH"])
	}
}
