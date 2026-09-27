package cmd

import (
	"github.com/yjwong/lark-cli/internal/desktop"
	"testing"
)

func TestDesktopInputValidation(t *testing.T) {
	spec := desktop.ReadOperation{Parameters: []desktop.OperationParameter{{Name: "id", Type: "string", Required: true}, {Name: "count", Type: "integer"}, {Name: "enabled", Type: "boolean"}, {Name: "ids", Type: "string[]"}}}
	for _, input := range []map[string]interface{}{{}, {"id": nil}, {"id": "x", "extra": true}, {"id": "x", "count": 1.5}, {"id": "x", "count": 1e30}, {"id": "x", "enabled": "true"}, {"id": "x", "ids": []interface{}{float64(1)}}} {
		if validateDesktopInputs(spec, input) == nil {
			t.Errorf("accepted invalid input %#v", input)
		}
	}
	if err := validateDesktopInputs(spec, map[string]interface{}{"id": "x", "enabled": true, "count": float64(2), "ids": []interface{}{"a"}}); err != nil {
		t.Fatal(err)
	}
}
