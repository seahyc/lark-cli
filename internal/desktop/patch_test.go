package desktop

import (
	"encoding/binary"
	"encoding/json"
	"os"
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

func TestSupportedMailArchiveFailsClosed(t *testing.T) {
	if supportedMailArchive("") || supportedMailArchive("0000") {
		t.Fatal("unknown archive accepted")
	}
	for hash := range supportedMailArchiveHashes {
		if len(hash) != 64 || !supportedMailArchive(hash) {
			t.Fatalf("malformed supported hash %q", hash)
		}
	}
}

func TestPatchInstalledArchiveWhenSupported(t *testing.T) {
	path := "/Applications/LarkSuite.app/Contents/Frameworks/Lark Framework.framework/Versions/Current/Resources/webcontent/mail.asar"
	original, err := os.ReadFile(path)
	if err != nil || !supportedMailArchive(digest(original)) {
		t.Skip("no supported installed Lark mail archive")
	}
	modules, err := moduleJavaScript()
	if err != nil {
		t.Fatal(err)
	}
	mutations, err := mutationJavaScript()
	if err != nil {
		t.Fatal(err)
	}
	script := strings.Replace(bridgeScript, "__LARK_LOCAL_SESSION__", "{}", 1)
	script = strings.Replace(script, "__LARK_READ_OPERATIONS__", modules, 1)
	script = strings.Replace(script, "__LARK_MUTATION_OPERATIONS__", mutations, 1)
	patched, err := patchASAR(original, script)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patched), "mail.card-attachment-download") || !strings.Contains(string(patched), "__larkLocalMailBridge") {
		t.Fatal("patched archive is missing the bridge or the new operations")
	}
}
