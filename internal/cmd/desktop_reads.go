package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/desktop"
)

func init() {
	ops := &cobra.Command{Use: "operations", Short: "List curated schema-verified native reads and their inputs (live status in domain reports)", RunE: func(cmd *cobra.Command, args []string) error {
		o, e := desktop.ReadOperations()
		if e != nil {
			return e
		}
		return desktopJSON(cmd, o)
	}}
	var input string
	run := &cobra.Command{Use: "run OPERATION", Short: "Execute one curated native read in the current desktop session", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if len(input) > 65536 {
			return fmt.Errorf("input exceeds 64 KiB")
		}
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(input), &params); err != nil || params == nil {
			return fmt.Errorf("--input must be a JSON object")
		}
		ops, err := desktop.ReadOperations()
		if err != nil {
			return err
		}
		var spec *desktop.ReadOperation
		for i := range ops {
			if ops[i].Name == args[0] {
				spec = &ops[i]
				break
			}
		}
		if spec == nil {
			return fmt.Errorf("unknown curated read %q; use desktop operations", args[0])
		}
		if err := validateDesktopInputs(*spec, params); err != nil {
			return err
		}
		var out json.RawMessage
		if err = desktopCall(cmd, args[0], params, &out); err != nil {
			return err
		}
		return desktopJSON(cmd, out)
	}}
	run.Flags().StringVar(&input, "input", "{}", "JSON object containing only documented operation inputs")
	desktopCmd.AddCommand(ops, run)
}

func validateDesktopInputs(spec desktop.ReadOperation, params map[string]interface{}) error {
	allowed := map[string]bool{}
	for _, p := range spec.Parameters {
		allowed[p.Name] = true
		v, has := params[p.Name]
		if p.Required && !has {
			return fmt.Errorf("missing required input %s", p.Name)
		}
		if !has {
			continue
		}
		valid := false
		switch p.Type {
		case "string":
			_, valid = v.(string)
		case "boolean":
			_, valid = v.(bool)
		case "integer":
			n, ok := v.(float64)
			valid = ok && n == float64(int64(n)) && n >= -9007199254740991 && n <= 9007199254740991
		case "string[]":
			a, ok := v.([]interface{})
			valid = ok && len(a) <= 100
			for _, x := range a {
				if _, ok := x.(string); !ok {
					valid = false
				}
			}
		default:
			return fmt.Errorf("unsupported parameter type %s", p.Type)
		}
		if !valid {
			return fmt.Errorf("%s must be %s", p.Name, p.Type)
		}
	}
	for k := range params {
		if !allowed[k] {
			return fmt.Errorf("unknown input %s", k)
		}
	}
	return nil
}
