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
	return filepath.Join(d, "lark-cli", "desktop"), e
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
func atomicWrite(path string, b []byte, mode os.FileMode) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".lark-cli-*")
	if e != nil {
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
	if _, e = os.Stat(filepath.Join(d, "session.json")); !os.IsNotExist(e) {
		return s, fmt.Errorf("session exists; restore it first")
	}
	path, e := filepath.EvalSymlinks("/Applications/LarkSuite.app/Contents/Frameworks/Lark Framework.framework/Versions/Current/Resources/webcontent/mail.asar")
	if e != nil {
		return s, e
	}
	original, e := os.ReadFile(path)
	if e != nil {
		return s, e
	}
	// Fail closed after a Lark update until its native protocol is revalidated.
	if digest(original) != "fd2d495a7d8f4da81334a3695cdc996060aa16f7c20cd529d2c53c683e2f5c8c" {
		return s, fmt.Errorf("unsupported Lark mail archive; revalidate the native protocol before patching this version")
	}
	nonce := make([]byte, 32)
	if _, e = rand.Read(nonce); e != nil {
		return s, e
	}
	s = Session{Port: port, Token: hex.EncodeToString(nonce), ExpiresAt: time.Now().Add(30 * time.Minute).UnixMilli(), Archive: path, OriginalHash: digest(original)}
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
	if e = atomicWrite(filepath.Join(d, "mail.asar.original"), original, 0600); e != nil {
		return s, e
	}
	cfg, _ = json.Marshal(s)
	if e = atomicWrite(filepath.Join(d, "session.json"), cfg, 0600); e != nil {
		return s, e
	}
	e = atomicWrite(path, patched, 0644)
	if e != nil {
		_ = os.Remove(filepath.Join(d, "session.json"))
	}
	return s, e
}
func RestoreSession() error {
	s, e := ReadSession()
	if e != nil {
		return e
	}
	d, _ := SessionDir()
	original, e := os.ReadFile(filepath.Join(d, "mail.asar.original"))
	if e != nil {
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
	return os.Remove(filepath.Join(d, "session.json"))
}
