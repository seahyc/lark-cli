package desktop

import (
	"runtime"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	platform, err := DetectPlatform()
	if err != nil {
		t.Skipf("Platform detection failed (expected on CI without Lark installed): %v", err)
	}
	
	if platform.AppDir == "" {
		t.Error("Expected non-empty AppDir")
	}
	if platform.AsarPath == "" {
		t.Error("Expected non-empty AsarPath")
	}
	if platform.UserDataDir == "" {
		t.Error("Expected non-empty UserDataDir")
	}
	if len(platform.LauncherPaths) == 0 {
		t.Error("Expected at least one launcher path")
	}
}

func TestIsMailTarget(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
		goos     string
	}{
		{
			name:     "macOS mail path",
			url:      "file:///Applications/LarkSuite.app/Contents/Frameworks/Lark Framework.framework/Versions/A/Resources/webcontent/mail/mail/index.html",
			expected: runtime.GOOS == "darwin",
			goos:     "darwin",
		},
		{
			name:     "macOS AutoFilterDialog path",
			url:      "file:///Applications/LarkSuite.app/Contents/Resources/webcontent/mail/AutoFilterDialog/index.html",
			expected: runtime.GOOS == "darwin",
			goos:     "darwin",
		},
		{
			name:     "Linux mail path",
			url:      "file:///opt/bytedance/lark/webcontent/mail/mail/index.html",
			expected: runtime.GOOS == "linux",
			goos:     "linux",
		},
		{
			name:     "Linux AutoFilterDialog path",
			url:      "file:///opt/bytedance/lark/webcontent/mail/AutoFilterDialog/index.html",
			expected: runtime.GOOS == "linux",
			goos:     "linux",
		},
		{
			name:     "wrong scheme",
			url:      "https://example.com/mail/index.html",
			expected: false,
		},
		{
			name:     "wrong path",
			url:      "file:///random/path/index.html",
			expected: false,
		},
		{
			name:     "empty url",
			url:      "",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.goos != "" && tt.goos != runtime.GOOS {
				t.Skipf("Test is for %s, running on %s", tt.goos, runtime.GOOS)
			}
			result := IsMailTarget(tt.url)
			if result != tt.expected {
				t.Errorf("IsMailTarget(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestValidateAsarHash(t *testing.T) {
	tests := []struct {
		name          string
		hash          string
		allowedHashes []string
		expectError   bool
	}{
		{
			name:          "exact match",
			hash:          "fd2d495a7d8f4da81334a3695cdc996060aa16f7c20cd529d2c53c683e2f5c8c",
			allowedHashes: []string{"fd2d495a7d8f4da81334a3695cdc996060aa16f7c20cd529d2c53c683e2f5c8c"},
			expectError:   false,
		},
		{
			name:          "Linux 7.72.23 hash",
			hash:          "7f2b8c498076c34d93e94f8d8511f8110c0bd61cc3311a8bbe570c7e0c2b51fd",
			allowedHashes: []string{"7f2b8c498076c34d93e94f8d8511f8110c0bd61cc3311a8bbe570c7e0c2b51fd"},
			expectError:   false,
		},
		{
			name:          "no match",
			hash:          "abc123",
			allowedHashes: []string{"def456"},
			expectError:   true,
		},
		{
			name:          "empty allowlist",
			hash:          "abc123",
			allowedHashes: []string{},
			expectError:   true,
		},
		{
			name:          "multiple allowed, match second",
			hash:          "7f2b8c498076c34d93e94f8d8511f8110c0bd61cc3311a8bbe570c7e0c2b51fd",
			allowedHashes: []string{"fd2d495a7d8f4da81334a3695cdc996060aa16f7c20cd529d2c53c683e2f5c8c", "7f2b8c498076c34d93e94f8d8511f8110c0bd61cc3311a8bbe570c7e0c2b51fd"},
			expectError:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAsarHash(tt.hash, tt.allowedHashes)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateAsarHash() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}
