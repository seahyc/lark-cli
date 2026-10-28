package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	MessageLinks struct {
		Browser      string `mapstructure:"browser"`
		WorkspaceURL string `mapstructure:"workspace_url"`
	} `mapstructure:"message_links"`
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
	Defaults  struct {
		Timezone        string `mapstructure:"timezone"`
		ReminderMinutes int    `mapstructure:"reminder_minutes"`
	} `mapstructure:"defaults"`
	OAuth struct {
		RedirectPort int `mapstructure:"redirect_port"`
	} `mapstructure:"oauth"`
	CustomEmojis map[string]string `mapstructure:"custom_emojis"`
}

var (
	cfg     *Config
	cfgDir  string
	rootDir string
)

// GetConfigDir returns the .lark directory path
func GetConfigDir() string {
	return cfgDir
}

// GetRootDir returns the project root directory
func GetRootDir() string {
	return rootDir
}

// Init initializes the configuration
func Init() error {
	// Config directory can be set via LARK_CONFIG_DIR or legacy LARK_CAL_CONFIG_DIR
	cfgDir = os.Getenv("LARK_CONFIG_DIR")
	if cfgDir == "" {
		cfgDir = os.Getenv("LARK_CAL_CONFIG_DIR") // Legacy fallback
	}
	if cfgDir == "" {
		return fmt.Errorf("LARK_CONFIG_DIR environment variable is not set")
	}

	rootDir = filepath.Dir(cfgDir)

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(cfgDir)

	// Set defaults
	viper.SetDefault("defaults.timezone", "Asia/Singapore")
	viper.SetDefault("defaults.reminder_minutes", 15)
	viper.SetDefault("oauth.redirect_port", 9999)

	// Environment variable bindings
	viper.SetEnvPrefix("LARK")
	viper.BindEnv("app_id", "LARK_APP_ID")
	viper.BindEnv("app_secret", "LARK_APP_SECRET")

	// Read config file (if exists)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config: %w", err)
		}
		// Config file not found is OK, we'll use defaults and env vars
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// Get returns the current configuration
func Get() *Config {
	if cfg == nil {
		cfg = &Config{}
	}
	return cfg
}

// GetAppID returns the app ID from config or environment
func GetAppID() string {
	return viper.GetString("app_id")
}

// GetAppSecret returns the app secret from environment
func GetAppSecret() string {
	return viper.GetString("app_secret")
}

// GetTimezone returns the default timezone
func GetTimezone() string {
	return viper.GetString("defaults.timezone")
}

// LoadTimezone loads the configured timezone as a *time.Location.
// Falls back to time.Local if the configured timezone is invalid.
func LoadTimezone() *time.Location {
	tz := GetTimezone()
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Local
	}
	return loc
}

// GetRedirectPort returns the OAuth redirect port
func GetRedirectPort() int {
	return viper.GetInt("oauth.redirect_port")
}

// TokensFilePath returns the path to the tokens file
func TokensFilePath() string {
	return filepath.Join(cfgDir, "tokens.json")
}

// TenantTokensFilePath returns the path to the tenant tokens file
func TenantTokensFilePath() string {
	return filepath.Join(cfgDir, "tenant_tokens.json")
}

// GetCustomEmojis returns the custom emoji mappings
func GetCustomEmojis() map[string]string {
	return viper.GetStringMapString("custom_emojis")
}

// ConfigFilePath returns the path of the YAML config file inside the config directory.
func ConfigFilePath() string {
	return filepath.Join(cfgDir, "config.yaml")
}

// readFileValues loads only the values stored in the config file (no defaults or env).
func readFileValues() (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(ConfigFilePath())
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || os.IsNotExist(err) {
			return v, nil
		}
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	return v, nil
}

// GetFileValue returns a value stored in the config file for key ("" if unset).
func GetFileValue(key string) (string, error) {
	v, err := readFileValues()
	if err != nil {
		return "", err
	}
	return v.GetString(key), nil
}

// setMu serialises in-process writers (viper's global state is not goroutine safe).
var setMu sync.Mutex

// otherConfigFiles are config names viper would also pick up for the "config" base
// name; creating config.yaml beside one of them would shadow or be shadowed by it.
var otherConfigFiles = []string{"config.yml", "config.json", "config.toml"}

// setYAMLValue sets the dotted key path to a string value in the YAML document,
// creating intermediate mappings and preserving all other nodes, comments and key
// case. Existing keys are matched case-insensitively (as viper reads them). When
// onlyIfUnset is true an existing non-empty scalar is left alone. It reports
// whether the document changed.
func setYAMLValue(doc *yaml.Node, key, value string, onlyIfUnset bool) (bool, error) {
	if doc.Kind == 0 {
		doc.Kind = yaml.DocumentNode
	}
	if doc.Kind != yaml.DocumentNode {
		return false, fmt.Errorf("unexpected config document structure")
	}
	if len(doc.Content) == 0 {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	node := doc.Content[0]
	if node.Kind == yaml.ScalarNode && node.Tag == "!!null" {
		*node = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", HeadComment: node.HeadComment, LineComment: node.LineComment, FootComment: node.FootComment}
	}
	if node.Kind != yaml.MappingNode {
		return false, fmt.Errorf("config file root is not a mapping")
	}
	parts := strings.Split(key, ".")
	for i, part := range parts {
		var child *yaml.Node
		for j := 0; j+1 < len(node.Content); j += 2 {
			if strings.EqualFold(node.Content[j].Value, part) {
				child = node.Content[j+1]
				break
			}
		}
		last := i == len(parts)-1
		if child == nil {
			if last {
				child = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
			} else {
				child = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			}
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part}, child)
			if last {
				return true, nil
			}
		} else if last {
			if onlyIfUnset && child.Kind == yaml.ScalarNode && child.Tag != "!!null" && child.Value != "" {
				return false, nil
			}
			if child.Kind != yaml.ScalarNode && child.Kind != yaml.AliasNode {
				return false, fmt.Errorf("config key %q is not a scalar value", key)
			}
			*child = yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value, HeadComment: child.HeadComment, LineComment: child.LineComment, FootComment: child.FootComment}
			return true, nil
		} else if child.Kind == yaml.ScalarNode && child.Tag == "!!null" {
			*child = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", HeadComment: child.HeadComment, LineComment: child.LineComment, FootComment: child.FootComment}
		} else if child.Kind != yaml.MappingNode {
			return false, fmt.Errorf("config key %q is not a mapping", strings.Join(parts[:i+1], "."))
		}
		node = child
	}
	return false, nil
}

// SetFileValue persists key=value into the config file in the config directory,
// preserving other file content, and updates the in-process configuration.
func SetFileValue(key, value string) error {
	_, err := setFileValue(key, value, false)
	return err
}

// SetFileValueIfUnset persists key=value only when the config file has no value
// for key. It reports whether a value was written.
func SetFileValueIfUnset(key, value string) (bool, error) {
	return setFileValue(key, value, true)
}

func setFileValue(key, value string, onlyIfUnset bool) (bool, error) {
	if cfgDir == "" {
		return false, fmt.Errorf("config directory is not initialised")
	}
	setMu.Lock()
	defer setMu.Unlock()
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		return false, fmt.Errorf("failed to create config directory: %w", err)
	}
	for _, name := range otherConfigFiles {
		if _, err := os.Stat(filepath.Join(cfgDir, name)); err == nil {
			return false, fmt.Errorf("found %s in %s; only config.yaml can be written by `lark config set` - merge it into config.yaml and remove %s", name, cfgDir, name)
		}
	}
	unlock, err := lockDir(cfgDir)
	if err != nil {
		return false, fmt.Errorf("failed to lock config: %w", err)
	}
	defer unlock()

	path := ConfigFilePath()
	var doc yaml.Node
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("error reading config: %w", err)
	}
	if err == nil {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return false, fmt.Errorf("error parsing %s (not modified): %w", path, err)
		}
	}
	changed, err := setYAMLValue(&doc, key, value, onlyIfUnset)
	if err != nil {
		return false, fmt.Errorf("cannot update %s: %w", path, err)
	}
	if !changed {
		return false, nil
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return false, fmt.Errorf("failed to encode config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return false, fmt.Errorf("failed to encode config: %w", err)
	}
	if err := writeFileAtomic(path, buf.Bytes()); err != nil {
		return false, err
	}

	// Refresh in-process config from the file just written. The value is already
	// persisted, so a refresh failure is not reported as a failed write.
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err == nil {
		next := &Config{}
		if viper.Unmarshal(next) == nil {
			cfg = next
		}
	}
	return true, nil
}

// writeFileAtomic writes data to a 0600 temp file in the same directory and renames it over path.
func writeFileAtomic(path string, data []byte) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	tmp := f.Name()
	defer func() {
		if err != nil {
			os.Remove(tmp)
		}
	}()
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}
