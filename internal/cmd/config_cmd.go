package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/config"
	"github.com/yjwong/lark-cli/internal/output"
)

// settableConfigKeys maps supported config keys to their value validators
// (returning the normalised value to store).
var settableConfigKeys = map[string]func(string) (string, error){
	"message_links.browser": func(v string) (string, error) {
		v = strings.ToLower(strings.TrimSpace(v))
		if !supportedLinkBrowser(v) {
			return "", fmt.Errorf("browser %q is not supported; use chrome, edge, brave, chromium, arc, or dia", v)
		}
		return v, nil
	},
	"message_links.workspace_url": func(v string) (string, error) {
		return linkWorkspaceOrigin(strings.TrimSpace(v))
	},
}

func settableKeyList() string {
	keys := make([]string, 0, len(settableConfigKeys))
	for k := range settableConfigKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// setConfigValue validates and persists a supported config key; it returns the stored value.
func setConfigValue(key, value string) (string, error) {
	validate, ok := settableConfigKeys[key]
	if !ok {
		return "", fmt.Errorf("unsupported config key %q; supported keys: %s", key, settableKeyList())
	}
	normalised, err := validate(value)
	if err != nil {
		return "", err
	}
	if err := config.SetFileValue(key, normalised); err != nil {
		return "", err
	}
	return normalised, nil
}

func getConfigValue(key string) (string, error) {
	if _, ok := settableConfigKeys[key]; !ok {
		return "", fmt.Errorf("unsupported config key %q; supported keys: %s", key, settableKeyList())
	}
	return config.GetFileValue(key)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Read and write CLI settings",
	Long:  "Read and write CLI settings stored in the config file. Supported keys: message_links.browser, message_links.workspace_url.",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value (e.g. message_links.workspace_url https://<team>.larksuite.com)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		v, err := setConfigValue(args[0], args[1])
		if err != nil {
			output.Fatal("VALIDATION_ERROR", err)
		}
		output.JSON(map[string]string{"key": args[0], "value": v})
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print a config value from the config file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		v, err := getConfigValue(args[0])
		if err != nil {
			output.Fatal("VALIDATION_ERROR", err)
		}
		output.JSON(map[string]string{"key": args[0], "value": v})
	},
}

func init() {
	configCmd.AddCommand(configSetCmd, configGetCmd)
	rootCmd.AddCommand(configCmd)
}
