package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/desktop"
)

func init() {
	list := &cobra.Command{Use: "mutations", Short: "List curated native writes and their effects", RunE: func(cmd *cobra.Command, args []string) error {
		ops, err := desktop.MutationOperations()
		if err != nil {
			return err
		}
		return desktopJSON(cmd, ops)
	}}
	var input, expectUser string
	var apply bool
	mutate := &cobra.Command{Use: "mutate OPERATION", Short: "Preview a typed native write; apply only with explicit identity", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if len(input) > 65536 {
			return fmt.Errorf("input exceeds 64 KiB")
		}
		var params map[string]interface{}
		if json.Unmarshal([]byte(input), &params) != nil || params == nil {
			return fmt.Errorf("--input must be a JSON object")
		}
		ops, err := desktop.MutationOperations()
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
			return fmt.Errorf("unknown curated mutation %q; use desktop mutations", args[0])
		}
		if err = validateDesktopInputs(*spec, params); err != nil {
			return err
		}
		if apply && expectUser == "" {
			return fmt.Errorf("--apply requires --expect-user from the preview")
		}
		var identity struct {
			UserID string `json:"userId"`
		}
		if err = desktopCall(cmd, "identity", nil, &identity); err != nil {
			return err
		}
		if expectUser != "" && expectUser != identity.UserID {
			return fmt.Errorf("signed-in identity differs from --expect-user")
		}
		var out json.RawMessage
		err = desktopCall(cmd, args[0], map[string]interface{}{"expectedUserId": identity.UserID, "input": params, "apply": apply}, &out)
		if err != nil && apply {
			return fmt.Errorf("%w; write outcome may be uncertain; inspect state before retrying", err)
		}
		if err != nil {
			return err
		}
		return desktopJSON(cmd, out)
	}}
	mutate.Flags().StringVar(&input, "input", "{}", "Typed JSON inputs listed by desktop mutations")
	mutate.Flags().BoolVar(&apply, "apply", false, "Execute the reviewed write once; outbound actions require recipient/content approval")
	mutate.Flags().StringVar(&expectUser, "expect-user", "", "Native user ID from preview")
	desktopCmd.AddCommand(list, mutate)
}
