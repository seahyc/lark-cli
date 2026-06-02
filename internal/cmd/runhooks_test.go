package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestEnableTraverseRunHooks guards the fix for the PersistentPreRun shadowing
// bug: grouped subcommands (msg, chat, mail, …) each define their own
// PersistentPreRun for scope validation. Without cobra.EnableTraverseRunHooks,
// cobra runs ONLY the closest hook, shadowing the root hook that resolves the
// global --format flag — which left output.Format at "pretty" for every grouped
// subcommand, silently dropping --format json/ndjson/text on e.g. `msg history`.
//
// The package init() sets cobra.EnableTraverseRunHooks = true. This test asserts
// that with it on, the full chain (root -> child) runs in order, so the root
// format hook fires before the child scope hook.
func TestEnableTraverseRunHooks(t *testing.T) {
	if !cobra.EnableTraverseRunHooks {
		t.Fatal("cobra.EnableTraverseRunHooks must be true so the root --format hook is not shadowed by grouped subcommand hooks")
	}

	var ran []string
	root := &cobra.Command{
		Use:           "root",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(*cobra.Command, []string) {
			ran = append(ran, "root")
		},
	}
	group := &cobra.Command{
		Use: "group",
		PersistentPreRun: func(*cobra.Command, []string) {
			ran = append(ran, "group")
		},
	}
	leaf := &cobra.Command{
		Use: "leaf",
		Run: func(*cobra.Command, []string) { ran = append(ran, "leaf") },
	}
	group.AddCommand(leaf)
	root.AddCommand(group)
	root.SetArgs([]string{"group", "leaf"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := []string{"root", "group", "leaf"}
	if len(ran) != len(want) {
		t.Fatalf("hooks ran = %v, want %v", ran, want)
	}
	for i := range want {
		if ran[i] != want[i] {
			t.Fatalf("hooks ran = %v, want %v (root must run before the group hook)", ran, want)
		}
	}
}
