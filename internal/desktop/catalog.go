package desktop

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed catalog.json
var catalogJSON []byte

type NativeDeclaration struct {
	Command  string `json:"command"`
	Name     string `json:"name"`
	Request  string `json:"request"`
	Response string `json:"response"`
	Status   string `json:"status"`
	Sources  []struct {
		Archive string `json:"archive"`
		File    string `json:"file"`
		Offset  int    `json:"offset"`
	} `json:"sources"`
}

func NativeCatalog(domain, search string) ([]NativeDeclaration, error) {
	var data struct {
		Operations []NativeDeclaration `json:"operations"`
	}
	if err := json.Unmarshal(catalogJSON, &data); err != nil {
		return nil, err
	}
	result := []NativeDeclaration{}
	for _, op := range data.Operations {
		if domain != "" && strings.Split(op.Request, ".")[0] != domain {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(op.Name+" "+op.Request), strings.ToLower(search)) {
			continue
		}
		result = append(result, op)
	}
	return result, nil
}
