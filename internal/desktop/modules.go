package desktop

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

//go:embed modules
var moduleFiles embed.FS

type OperationParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}
type ReadOperation struct {
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Transport    string               `json:"transport,omitempty"`
	ReadOnly     bool                 `json:"readOnly"`
	Effect       string               `json:"effect,omitempty"`
	Parameters   []OperationParameter `json:"parameters"`
	Evidence     []json.RawMessage    `json:"evidence"`
	Verification string               `json:"verification"`
}

func readModules() (map[string]ReadOperation, []string, error) { return loadModules(false) }
func loadModules(mutations bool) (map[string]ReadOperation, []string, error) {
	ops := map[string]ReadOperation{}
	scripts := []string{}
	manifest, source := "manifest.json", "module.js"
	if mutations {
		manifest, source = "mutation-manifest.json", "mutations.js"
	}
	paths, err := fs.Glob(moduleFiles, "modules/*/"+manifest)
	if err != nil {
		return nil, nil, err
	}
	valid := regexp.MustCompile(`^[a-z][a-z0-9-]*\.[a-z][a-z0-9-]*$`)
	for _, path := range paths {
		data, err := moduleFiles.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		var entries []ReadOperation
		if err = json.Unmarshal(data, &entries); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		for _, entry := range entries {
			if entry.Transport != "" && entry.Transport != "sdk" && entry.Transport != "server" && !(entry.Transport == "shell-navigation" && !mutations && entry.Name == "workspacenext.navigation-layout") {
				return nil, nil, fmt.Errorf("invalid transport for %s", entry.Name)
			}
			if !valid.MatchString(entry.Name) || entry.ReadOnly == mutations || entry.Verification != "schema-verified" || len(entry.Evidence) == 0 {
				return nil, nil, fmt.Errorf("invalid read operation manifest: %s", path)
			}
			if mutations && entry.Effect != "settings" && entry.Effect != "draft" && entry.Effect != "outbound" {
				return nil, nil, fmt.Errorf("invalid mutation effect: %s", entry.Name)
			}
			if _, exists := ops[entry.Name]; exists {
				return nil, nil, fmt.Errorf("duplicate operation %s", entry.Name)
			}
			ops[entry.Name] = entry
		}
		script, err := moduleFiles.ReadFile(strings.TrimSuffix(path, manifest) + source)
		if err != nil {
			return nil, nil, err
		}
		prefix := ""
		if mutations && strings.Contains(path, "modules/triage/") {
			helper, err := moduleFiles.ReadFile("modules/triage/text_codec.js")
			if err != nil {
				return nil, nil, err
			}
			prefix = "const scheduleTextCodec=(()=>{const module={exports:{}};\n" + string(helper) + "\nreturn module.exports;})();\n"
		}
		scripts = append(scripts, "(()=>{const module={exports:{}};\n"+prefix+string(script)+"\nreturn module.exports.operations;})()")
	}
	return ops, scripts, nil
}
func ReadOperations() ([]ReadOperation, error) {
	ops, _, err := readModules()
	if err != nil {
		return nil, err
	}
	result := make([]ReadOperation, 0, len(ops))
	for _, op := range ops {
		result = append(result, op)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}
func readOperationExists(name string) bool {
	ops, _, err := readModules()
	if err != nil {
		return false
	}
	_, ok := ops[name]
	return ok
}
func moduleJavaScript() (string, error) {
	_, scripts, err := readModules()
	if err != nil {
		return "", err
	}
	return "Object.assign({}," + strings.Join(append([]string{"{}"}, scripts...), ",") + ")", nil
}

func MutationOperations() ([]ReadOperation, error) {
	ops, _, err := loadModules(true)
	if err != nil {
		return nil, err
	}
	result := make([]ReadOperation, 0, len(ops))
	for _, op := range ops {
		result = append(result, op)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}
func mutationOperationExists(name string) bool {
	ops, _, err := loadModules(true)
	if err != nil {
		return false
	}
	_, ok := ops[name]
	return ok
}
func mutationJavaScript() (string, error) {
	_, scripts, err := loadModules(true)
	if err != nil {
		return "", err
	}
	return "Object.assign({}," + strings.Join(append([]string{"{}"}, scripts...), ",") + ")", nil
}
