package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var wikiCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Manage Lark Wiki spaces and nodes",
	Long:  "List wiki spaces, browse and manage wiki nodes (pages).",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		validateScopeGroup("documents")
	},
}

// --- wiki spaces ---

var wikiSpacesCmd = &cobra.Command{
	Use:   "spaces",
	Short: "List wiki spaces",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		spaces, err := client.ListWikiSpaces()
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"spaces": spaces,
			"count":  len(spaces),
		})
	},
}

// --- wiki space ---

var wikiSpaceGetCmd = &cobra.Command{
	Use:   "space <space-id>",
	Short: "Get a wiki space's metadata",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		space, err := client.GetWikiSpace(args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(space)
	},
}

// --- wiki create ---

var (
	wikiCreateSpaceID    string
	wikiCreateTitle      string
	wikiCreateParent     string
	wikiCreateObjType    string
	wikiCreateNodeType   string
	wikiCreateOriginNode string
)

var wikiCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a wiki node (page)",
	Long: `Create a new wiki node inside a space.

--obj-type defaults to "docx" and accepts: doc, docx, sheet, bitable, mindnote,
file, slides.

--node-type defaults to "origin". Use "shortcut" with --origin-node to link to an
existing node.

Examples:
  lark wiki create --space-id 7234... --title "Design Notes"
  lark wiki create --space-id 7234... --title "Tracker" --obj-type bitable
  lark wiki create --space-id 7234... --parent nodeTokenABC --title "Sub page"`,
	Run: func(cmd *cobra.Command, args []string) {
		if wikiCreateSpaceID == "" {
			output.Fatalf("VALIDATION_ERROR", "--space-id is required")
		}
		objType := wikiCreateObjType
		if objType == "" {
			objType = "docx"
		}
		nodeType := wikiCreateNodeType
		if nodeType == "" {
			nodeType = "origin"
		}
		req := &api.CreateWikiNodeRequest{
			ObjType:         objType,
			NodeType:        nodeType,
			ParentNodeToken: wikiCreateParent,
			OriginNodeToken: wikiCreateOriginNode,
			Title:           wikiCreateTitle,
		}
		client := api.NewClient()
		node, err := client.CreateWikiNode(wikiCreateSpaceID, req)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(node)
	},
}

// --- wiki move ---

var (
	wikiMoveSpaceID       string
	wikiMoveParent        string
	wikiMoveTargetSpaceID string
)

var wikiMoveCmd = &cobra.Command{
	Use:   "move <node-token>",
	Short: "Move a wiki node under a new parent",
	Long: `Move a wiki node to a new parent, optionally moving it to a different space.

Examples:
  lark wiki move nodeTokenXYZ --space-id 7234... --parent nodeTokenParent
  lark wiki move nodeTokenXYZ --space-id 7234... --target-space-id 7999...`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if wikiMoveSpaceID == "" {
			output.Fatalf("VALIDATION_ERROR", "--space-id is required")
		}
		if wikiMoveParent == "" && wikiMoveTargetSpaceID == "" {
			output.Fatalf("VALIDATION_ERROR", "--parent or --target-space-id is required")
		}
		client := api.NewClient()
		node, err := client.MoveWikiNode(wikiMoveSpaceID, args[0], wikiMoveParent, wikiMoveTargetSpaceID)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(node)
	},
}

// --- wiki delete ---

var wikiDeleteSpaceID string

var wikiDeleteCmd = &cobra.Command{
	Use:   "delete <node-token>",
	Short: "Delete a wiki node",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if wikiDeleteSpaceID == "" {
			output.Fatalf("VALIDATION_ERROR", "--space-id is required")
		}
		client := api.NewClient()
		if err := client.DeleteWikiNode(wikiDeleteSpaceID, args[0]); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":    true,
			"space_id":   wikiDeleteSpaceID,
			"node_token": args[0],
		})
	},
}

// --- wiki get ---

var wikiGetCmd = &cobra.Command{
	Use:   "get <node-token>",
	Short: "Get a wiki node's metadata",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		node, err := client.GetWikiNode(args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(api.OutputWikiNode{
			NodeToken: node.NodeToken,
			ObjToken:  node.ObjToken,
			ObjType:   node.ObjType,
			Title:     node.Title,
			SpaceID:   node.SpaceID,
			NodeType:  node.NodeType,
			HasChild:  node.HasChild,
		})
	},
}

// --- wiki children ---

var wikiChildrenCmd = &cobra.Command{
	Use:   "children <node-token>",
	Short: "List child nodes of a wiki node",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		node, err := client.GetWikiNode(args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		children, err := client.GetWikiNodeChildren(node.SpaceID, args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		out := make([]api.OutputWikiNode, len(children))
		for i, ch := range children {
			out[i] = api.OutputWikiNode{
				NodeToken: ch.NodeToken,
				ObjToken:  ch.ObjToken,
				ObjType:   ch.ObjType,
				Title:     ch.Title,
				SpaceID:   ch.SpaceID,
				NodeType:  ch.NodeType,
				HasChild:  ch.HasChild,
			}
		}
		output.JSON(api.OutputWikiChildren{
			ParentNodeToken: args[0],
			SpaceID:         node.SpaceID,
			Children:        out,
			Count:           len(out),
		})
	},
}

func init() {
	wikiCreateCmd.Flags().StringVar(&wikiCreateSpaceID, "space-id", "", "Wiki space ID (required)")
	wikiCreateCmd.Flags().StringVar(&wikiCreateTitle, "title", "", "Node title")
	wikiCreateCmd.Flags().StringVar(&wikiCreateParent, "parent", "", "Parent node token (top-level if omitted)")
	wikiCreateCmd.Flags().StringVar(&wikiCreateObjType, "obj-type", "", "Object type: docx (default), doc, sheet, bitable, mindnote, file, slides")
	wikiCreateCmd.Flags().StringVar(&wikiCreateNodeType, "node-type", "", "Node type: origin (default) or shortcut")
	wikiCreateCmd.Flags().StringVar(&wikiCreateOriginNode, "origin-node", "", "Origin node token (required when --node-type=shortcut)")

	wikiMoveCmd.Flags().StringVar(&wikiMoveSpaceID, "space-id", "", "Current space ID (required)")
	wikiMoveCmd.Flags().StringVar(&wikiMoveParent, "parent", "", "New parent node token")
	wikiMoveCmd.Flags().StringVar(&wikiMoveTargetSpaceID, "target-space-id", "", "Target space ID (for cross-space moves)")

	wikiDeleteCmd.Flags().StringVar(&wikiDeleteSpaceID, "space-id", "", "Wiki space ID (required)")

	wikiCmd.AddCommand(wikiSpacesCmd)
	wikiCmd.AddCommand(wikiSpaceGetCmd)
	wikiCmd.AddCommand(wikiCreateCmd)
	wikiCmd.AddCommand(wikiMoveCmd)
	wikiCmd.AddCommand(wikiDeleteCmd)
	wikiCmd.AddCommand(wikiGetCmd)
	wikiCmd.AddCommand(wikiChildrenCmd)
}
