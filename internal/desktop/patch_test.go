package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChownDirTree(t *testing.T) {
	if os.Getenv("SUDO_USER") == "" || os.Geteuid() != 0 {
		t.Skip("Test requires sudo environment")
	}
	
	tmpHome := t.TempDir()
	targetDir := filepath.Join(tmpHome, ".cache", "lark-cli", "desktop")
	
	visited := make(map[string]bool)
	mockChown := func(path string) error {
		visited[path] = true
		return nil
	}
	
	if err := chownDirTree(targetDir, mockChown); err != nil {
		t.Fatalf("chownDirTree failed: %v", err)
	}
	
	expected := []string{
		filepath.Join(tmpHome, ".cache"),
		filepath.Join(tmpHome, ".cache", "lark-cli"),
		filepath.Join(tmpHome, ".cache", "lark-cli", "desktop"),
	}
	
	for _, path := range expected {
		if !visited[path] {
			t.Errorf("Expected chown to be called on %s, but it wasn't", path)
		}
	}
}

func TestChownDirTreeNonSudo(t *testing.T) {
	origSudoUser := os.Getenv("SUDO_USER")
	defer func() {
		if origSudoUser != "" {
			os.Setenv("SUDO_USER", origSudoUser)
		} else {
			os.Unsetenv("SUDO_USER")
		}
	}()
	
	os.Unsetenv("SUDO_USER")
	
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".cache", "test")
	
	called := false
	mockChown := func(path string) error {
		called = true
		return nil
	}
	
	if err := chownDirTree(targetDir, mockChown); err != nil {
		t.Fatalf("chownDirTree failed: %v", err)
	}
	
	if called {
		t.Error("chown should not be called when SUDO_USER is not set")
	}
}
