package desktop

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchASARKeepsOriginalContent(t *testing.T) {
	html := []byte("<html><body>original</body></html>")
	h, _ := json.Marshal(map[string]interface{}{"files": map[string]interface{}{"mail": map[string]interface{}{"files": map[string]interface{}{"en-US.html": map[string]interface{}{"offset": "0", "size": len(html)}}}}})
	n := (4 + len(h) + 3) &^ 3
	b := make([]byte, 12+n)
	binary.LittleEndian.PutUint32(b[0:4], 4)
	binary.LittleEndian.PutUint32(b[4:8], uint32(n+4))
	binary.LittleEndian.PutUint32(b[8:12], uint32(n))
	binary.LittleEndian.PutUint32(b[12:16], uint32(len(h)))
	copy(b[16:], h)
	b = append(b, html...)
	out, e := patchASAR(b, "void 0;")
	if e != nil {
		t.Fatal(e)
	}
	base := 8 + int(binary.LittleEndian.Uint32(out[4:8]))
	if string(out[base:base+len(html)]) != string(html) {
		t.Fatal("original data changed")
	}
	if !strings.Contains(string(out[base+len(html):]), "<script>void 0;</script></body>") {
		t.Fatal("missing injection")
	}
	if _, e := patchASAR([]byte("invalid"), ""); e == nil {
		t.Fatal("invalid ASAR accepted")
	}
}

func TestChownDirTree(t *testing.T) {
	mockHome := t.TempDir()
	targetDir := filepath.Join(mockHome, ".cache", "lark-cli", "desktop")
	
	visited := make(map[string]bool)
	mockChown := func(path string) error {
		visited[path] = true
		return nil
	}
	
	origSudoUser := os.Getenv("SUDO_USER")
	origChownFunc := chownToSudoUserFunc
	origLookupFunc := userLookupFunc
	origGetEuidFunc := getEuidFunc
	defer func() {
		if origSudoUser != "" {
			os.Setenv("SUDO_USER", origSudoUser)
		} else {
			os.Unsetenv("SUDO_USER")
		}
		chownToSudoUserFunc = origChownFunc
		userLookupFunc = origLookupFunc
		getEuidFunc = origGetEuidFunc
	}()
	
	os.Setenv("SUDO_USER", "mockuser")
	getEuidFunc = func() int { return 0 }
	
	userLookupFunc = func(username string) (*user.User, error) {
		if username == "mockuser" {
			return &user.User{HomeDir: mockHome}, nil
		}
		return nil, fmt.Errorf("user not found")
	}
	
	chownToSudoUserFunc = func(path string) error {
		if os.Getenv("SUDO_USER") == "" {
			return nil
		}
		return mockChown(path)
	}
	
	if err := chownDirTree(targetDir, chownToSudoUser); err != nil {
		t.Fatalf("chownDirTree failed: %v", err)
	}
	
	expected := []string{
		filepath.Join(mockHome, ".cache"),
		filepath.Join(mockHome, ".cache", "lark-cli"),
		filepath.Join(mockHome, ".cache", "lark-cli", "desktop"),
	}
	
	for _, path := range expected {
		if !visited[path] {
			t.Errorf("Expected chown to be called on %s, but it wasn't", path)
		}
	}
	
	outsidePath := filepath.Join(t.TempDir(), "outside", "path")
	visited = make(map[string]bool)
	if err := chownDirTree(outsidePath, chownToSudoUser); err != nil {
		t.Fatalf("chownDirTree failed for outside path: %v", err)
	}
	if len(visited) > 0 {
		t.Error("chown should not be called for paths outside home")
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
