package desktop

import (
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed mail_bridge.js
var bridgeScript string

type Session struct {
	Port         int    `json:"port"`
	Token        string `json:"token"`
	ExpiresAt    int64  `json:"expiresAt"`
	Archive      string `json:"archive"`
	OriginalHash string `json:"originalHash"`
	PatchedHash  string `json:"patchedHash"`
}

func SessionDir() (string, error) {
	d, e := os.UserCacheDir()
	if e != nil {
		return "", e
	}
	sessionDir := filepath.Join(d, "lark-cli", "desktop")
	if os.Getenv("SUDO_USER") != "" && os.Geteuid() == 0 {
		originalUser := os.Getenv("SUDO_USER")
		usr, e := user.Lookup(originalUser)
		if e != nil {
			return "", fmt.Errorf("cannot resolve SUDO_USER %s: %w", originalUser, e)
		}
		sessionDir = filepath.Join(usr.HomeDir, ".cache", "lark-cli", "desktop")
	}
	return sessionDir, nil
}

func chownToSudoUser(path string) error {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" || os.Geteuid() != 0 {
		return nil
	}
	usr, e := user.Lookup(sudoUser)
	if e != nil {
		return fmt.Errorf("cannot resolve SUDO_USER %s: %w", sudoUser, e)
	}
	uid, _ := strconv.Atoi(usr.Uid)
	gid, _ := strconv.Atoi(usr.Gid)
	return os.Chown(path, uid, gid)
}
func ReadSession() (Session, error) {
	var s Session
	d, e := SessionDir()
	if e != nil {
		return s, e
	}
	b, e := os.ReadFile(filepath.Join(d, "session.json"))
	if e != nil {
		return s, e
	}
	e = json.Unmarshal(b, &s)
	return s, e
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func ComputeHash(b []byte) string { return digest(b) }
func atomicWrite(path string, b []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	f, e := os.CreateTemp(dir, ".lark-cli-*")
	if e != nil {
		if os.IsPermission(e) {
			return fmt.Errorf("cannot write to %s (permission denied); on Linux, run with sudo to patch system-owned Lark files, or set LARK_APP_DIR to a user-writable location: %w", dir, e)
		}
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(mode); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(name, path)
}
func patchASAR(b []byte, script string) ([]byte, error) {
	if len(b) < 16 {
		return nil, fmt.Errorf("invalid ASAR")
	}
	hs := int(binary.LittleEndian.Uint32(b[4:8]))
	js := int(binary.LittleEndian.Uint32(b[12:16]))
	base := 8 + hs
	if hs < 8 || js < 2 || 16+js > base || base > len(b) {
		return nil, fmt.Errorf("invalid ASAR header")
	}
	var header map[string]interface{}
	if e := json.Unmarshal(b[16:16+js], &header); e != nil {
		return nil, e
	}
	node := header
	// Only patches en-US.html; other locales remain unpatched.
	// Users must use English locale to access the bridge.
	for _, key := range []string{"mail", "en-US.html"} {
		files, ok := node["files"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing ASAR files")
		}
		node, ok = files[key].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing mail/en-US.html")
		}
	}
	offset, e := strconv.Atoi(fmt.Sprint(node["offset"]))
	if e != nil {
		return nil, e
	}
	sz, ok := node["size"].(float64)
	if !ok || offset < 0 || base+offset+int(sz) > len(b) {
		return nil, fmt.Errorf("invalid HTML entry")
	}
	html := string(b[base+offset : base+offset+int(sz)])
	if !strings.Contains(html, "</body>") || strings.Contains(html, "__larkLocalMailBridge") {
		return nil, fmt.Errorf("unexpected or already patched HTML")
	}
	replacement := []byte(strings.Replace(html, "</body>", "<script>"+script+"</script></body>", 1))
	node["offset"] = strconv.Itoa(len(b) - base)
	node["size"] = len(replacement)
	hash := digest(replacement)
	node["integrity"] = map[string]interface{}{"algorithm": "SHA256", "hash": hash, "blockSize": 4194304, "blocks": []string{hash}}
	h, e := json.Marshal(header)
	if e != nil {
		return nil, e
	}
	payload := (4 + len(h) + 3) &^ 3
	out := make([]byte, 12+payload)
	binary.LittleEndian.PutUint32(out[0:4], 4)
	binary.LittleEndian.PutUint32(out[4:8], uint32(4+payload))
	binary.LittleEndian.PutUint32(out[8:12], uint32(payload))
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(h)))
	copy(out[16:], h)
	out = append(out, b[base:]...)
	return append(out, replacement...), nil
}
func InstallSession(port int) (Session, error) {
	var s Session
	if port < 1024 || port > 65535 {
		return s, fmt.Errorf("invalid port")
	}
	d, e := SessionDir()
	if e != nil {
		return s, e
	}
	if e = os.MkdirAll(d, 0700); e != nil {
		return s, e
	}
	if e = chownToSudoUser(d); e != nil {
		return s, fmt.Errorf("chown session dir: %w", e)
	}
	if _, e = os.Stat(filepath.Join(d, "session.json")); !os.IsNotExist(e) {
		return s, fmt.Errorf("session exists; restore it first")
	}
	platform, e := DetectPlatform()
	if e != nil {
		return s, e
	}
	original, e := os.ReadFile(platform.AsarPath)
	if e != nil {
		return s, e
	}
	hash := digest(original)
	if e = ValidateAsarHash(hash, platform.AllowedHashes); e != nil {
		return s, e
	}
	nonce := make([]byte, 32)
	if _, e = rand.Read(nonce); e != nil {
		return s, e
	}
	s = Session{Port: port, Token: hex.EncodeToString(nonce), ExpiresAt: time.Now().Add(30 * time.Minute).UnixMilli(), Archive: platform.AsarPath, OriginalHash: hash}
	cfg, _ := json.Marshal(s)
	modules, e := moduleJavaScript()
	if e != nil {
		return s, e
	}
	script := strings.Replace(bridgeScript, "__LARK_LOCAL_SESSION__", string(cfg), 1)
	script = strings.Replace(script, "__LARK_READ_OPERATIONS__", modules, 1)
	mutations, e := mutationJavaScript()
	if e != nil {
		return s, e
	}
	script = strings.Replace(script, "__LARK_MUTATION_OPERATIONS__", mutations, 1)
	patched, e := patchASAR(original, script)
	if e != nil {
		return s, e
	}
	s.PatchedHash = digest(patched)
	originalPath := filepath.Join(d, "mail.asar.original")
	if e = atomicWrite(originalPath, original, 0600); e != nil {
		return s, e
	}
	if e = chownToSudoUser(originalPath); e != nil {
		return s, fmt.Errorf("chown backup: %w", e)
	}
	cfg, _ = json.Marshal(s)
	sessionPath := filepath.Join(d, "session.json")
	if e = atomicWrite(sessionPath, cfg, 0600); e != nil {
		return s, e
	}
	if e = chownToSudoUser(sessionPath); e != nil {
		return s, fmt.Errorf("chown session: %w", e)
	}
	e = atomicWrite(platform.AsarPath, patched, 0644)
	if e != nil {
		_ = os.Remove(filepath.Join(d, "session.json"))
	}
	return s, e
}
func RestoreSession() error {
	s, e := ReadSession()
	if e != nil {
		if os.IsPermission(e) && os.Getenv("SUDO_USER") != "" {
			return fmt.Errorf("session files are owned by root; run with sudo: %w", e)
		}
		return e
	}
	d, _ := SessionDir()
	originalPath := filepath.Join(d, "mail.asar.original")
	original, e := os.ReadFile(originalPath)
	if e != nil {
		if os.IsPermission(e) && os.Getenv("SUDO_USER") != "" {
			return fmt.Errorf("backup file is owned by root; run with sudo: %w", e)
		}
		return e
	}
	if digest(original) != s.OriginalHash {
		return fmt.Errorf("backup hash mismatch")
	}
	current, e := os.ReadFile(s.Archive)
	if e != nil {
		return e
	}
	hash := digest(current)
	if hash != s.OriginalHash && hash != s.PatchedHash {
		return fmt.Errorf("Lark resources changed; refusing to overwrite them")
	}
	if hash != s.OriginalHash {
		if e = atomicWrite(s.Archive, original, 0644); e != nil {
			return e
		}
	}
	sessionPath := filepath.Join(d, "session.json")
	if e = os.Remove(sessionPath); e != nil {
		return e
	}
	return os.Remove(originalPath)
}
