package desktop

import (
	"encoding/json"
	"os/exec"
	"testing"
)

// Exercise the embedded browser closure, where CommonJS require is unavailable.
// Module-only tests use Node require and cannot catch missing inline dependencies.
func TestEmbeddedScheduleCodecWithoutRequire(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is needed to execute the embedded browser JavaScript")
	}
	source, err := mutationJavaScript()
	if err != nil {
		t.Fatal(err)
	}
	quoted, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm=require('node:vm');const ops=vm.runInNewContext(` + string(quoted) + `);const d=ops['triage.schedule-self-text'];const stage=d.request({chatId:'123',text:'fixture',scheduleTime:'1800000000'});const wire=d.persistRequest(stage,{cid:stage.cid});const rich=wire.putRequest.content.richText;const prop=rich.elements.dictionary[rich.elementIds[0]].property;if(rich.innerText!=='fixture'||Array.from(prop).join(',')!=='10,7,102,105,120,116,117,114,101')throw Error('embedded server content differs');`
	cmd := exec.Command(node, "-e", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("embedded module failed: %v\n%s", err, output)
	}
}
