package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type PlatformConfig struct {
	AppDir         string
	AsarPath       string
	AllowedHashes  []string
	LauncherPaths  []string
	UserDataDir    string
}

func DetectPlatform() (*PlatformConfig, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectMacOS()
	case "linux":
		return detectLinux()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func detectMacOS() (*PlatformConfig, error) {
	appDir := "/Applications/LarkSuite.app"
	if override := os.Getenv("LARK_APP_DIR"); override != "" {
		appDir = override
	}

	asarPath := filepath.Join(appDir, "Contents/Frameworks/Lark Framework.framework/Versions/Current/Resources/webcontent/mail.asar")
	resolved, err := filepath.EvalSymlinks(asarPath)
	if err != nil {
		return nil, fmt.Errorf("macOS Lark mail.asar not found at %s: %w", asarPath, err)
	}

	return &PlatformConfig{
		AppDir:   appDir,
		AsarPath: resolved,
		AllowedHashes: []string{
			"fd2d495a7d8f4da81334a3695cdc996060aa16f7c20cd529d2c53c683e2f5c8c",
		},
		LauncherPaths: []string{
			filepath.Join(appDir, "Contents/MacOS/Lark"),
		},
		UserDataDir: filepath.Join(os.Getenv("HOME"), "Library/Application Support/LarkSuite"),
	}, nil
}

func detectLinux() (*PlatformConfig, error) {
	appDir := "/opt/bytedance/lark"
	if override := os.Getenv("LARK_APP_DIR"); override != "" {
		appDir = override
	}

	asarPath := filepath.Join(appDir, "webcontent/mail.asar")
	if _, err := os.Stat(asarPath); err != nil {
		return nil, fmt.Errorf("Linux Lark mail.asar not found at %s: %w", asarPath, err)
	}

	allowedHashes := []string{}
	if os.Getenv("LARK_LINUX_ALLOW_UNTRUSTED_ASAR") == "1" {
		allowedHashes = []string{
			"7f2b8c498076c34d93e94f8d8511f8110c0bd61cc3311a8bbe570c7e0c2b51fd",
		}
	}

	home := os.Getenv("HOME")
	if home == "" {
		return nil, fmt.Errorf("HOME environment variable not set")
	}

	return &PlatformConfig{
		AppDir:   appDir,
		AsarPath: asarPath,
		AllowedHashes: allowedHashes,
		LauncherPaths: []string{
			"/usr/bin/bytedance-lark-stable",
			filepath.Join(appDir, "lark"),
		},
		UserDataDir: filepath.Join(home, ".config/LarkInternational"),
	}, nil
}

func IsMailTarget(targetURL string) bool {
	u := targetURL
	if !strings.HasPrefix(u, "file://") {
		return false
	}
	
	path := strings.TrimPrefix(u, "file://")
	
	switch runtime.GOOS {
	case "darwin":
		return strings.HasPrefix(path, "/Applications/LarkSuite.app/Contents/") &&
			(strings.Contains(path, "/webcontent/mail/mail/") || 
			 strings.Contains(path, "/webcontent/mail/AutoFilterDialog/"))
	case "linux":
		return strings.HasPrefix(path, "/opt/bytedance/lark/") &&
			(strings.Contains(path, "/webcontent/mail/mail/") || 
			 strings.Contains(path, "/webcontent/mail/AutoFilterDialog/"))
	default:
		return false
	}
}

func ValidateAsarHash(hash string, allowedHashes []string) error {
	if len(allowedHashes) == 0 {
		return fmt.Errorf("unsupported Lark mail archive on %s; set LARK_LINUX_ALLOW_UNTRUSTED_ASAR=1 to proceed with unvalidated protocol (Linux only), or revalidate the native protocol before patching this version", runtime.GOOS)
	}
	
	for _, allowed := range allowedHashes {
		if hash == allowed {
			return nil
		}
	}
	
	hashPrefix := hash
	if len(hash) > 16 {
		hashPrefix = hash[:16] + "..."
	}
	return fmt.Errorf("unsupported Lark mail archive hash %s on %s; revalidate the native protocol before patching this version", hashPrefix, runtime.GOOS)
}
