package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"
)

const (
	DefaultAPIURL   = "https://wariwari.jp"
	LocalConfigFile = ".wariwari.yaml"
	serviceName     = "wariwari"
)

type Config struct {
	Token   string `yaml:"token" json:"-"`
	Profile string `yaml:"profile" json:"profile"`
	APIURL  string `yaml:"api_url" json:"api_url"`
	Group   string `yaml:"group" json:"group"`
}

type Profile struct {
	APIURL string `json:"api_url,omitempty"`
	Group  string `json:"group,omitempty"`
}

type ProfileStore struct {
	Default  string             `json:"default,omitempty"`
	Profiles map[string]Profile `json:"profiles,omitempty"`
	path     string
}

func Load() *Config {
	cfg := &Config{APIURL: DefaultAPIURL}

	for _, path := range GlobalConfigPaths() {
		if data, err := os.ReadFile(path); err == nil {
			_ = yaml.Unmarshal(data, cfg)
			break
		}
	}

	if localPath := LocalConfigPath(); localPath != "" {
		if data, err := os.ReadFile(localPath); err == nil {
			var local Config
			if yaml.Unmarshal(data, &local) == nil {
				merge(cfg, &local)
			}
		}
	}

	if v := strings.TrimSpace(os.Getenv("WARIWARI_TOKEN")); v != "" {
		cfg.Token = v
	}
	if v := strings.TrimSpace(os.Getenv("WARIWARI_PROFILE")); v != "" {
		cfg.Profile = v
	}
	if v := strings.TrimSpace(os.Getenv("WARIWARI_API_URL")); v != "" {
		cfg.APIURL = v
	}
	if v := strings.TrimSpace(os.Getenv("WARIWARI_GROUP")); v != "" {
		cfg.Group = v
	}

	EnsureAPIURL(cfg)
	return cfg
}

func LoadGlobal() *Config {
	cfg := &Config{APIURL: DefaultAPIURL}
	for _, path := range GlobalConfigPaths() {
		if data, err := os.ReadFile(path); err == nil {
			_ = yaml.Unmarshal(data, cfg)
			break
		}
	}
	EnsureAPIURL(cfg)
	return cfg
}

func merge(dst, src *Config) {
	if src.Token != "" {
		dst.Token = src.Token
	}
	if src.Profile != "" {
		dst.Profile = src.Profile
	}
	if src.APIURL != "" {
		dst.APIURL = src.APIURL
	}
	if src.Group != "" {
		dst.Group = src.Group
	}
}

func EnsureAPIURL(cfg *Config) {
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultAPIURL
	}
	cfg.APIURL = strings.TrimRight(cfg.APIURL, "/")
	if strings.HasPrefix(cfg.APIURL, "http://") {
		if u, err := url.Parse(cfg.APIURL); err == nil {
			host := u.Hostname()
			if host != "localhost" && host != "127.0.0.1" && host != "::1" {
				fmt.Fprintf(os.Stderr, "Warning: API URL %q uses HTTP. HTTPS is recommended.\n", cfg.APIURL)
			}
		}
	}
}

func GlobalConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".config", "wariwari", "config.yaml"),
		filepath.Join(home, ".wariwari", "config.yaml"),
	}
}

func ConfigPath() (string, error) {
	paths := GlobalConfigPaths()
	if len(paths) == 0 {
		return "", fmt.Errorf("unable to determine config path")
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, os.MkdirAll(filepath.Dir(path), 0o700)
		}
	}
	path := paths[0]
	return path, os.MkdirAll(filepath.Dir(path), 0o700)
}

func LocalConfigPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		path := filepath.Join(dir, LocalConfigFile)
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *Config) SaveLocal() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(LocalConfigFile, data, 0o600)
}

func ProfilesPath() (string, error) {
	cfgPath, err := ConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfgPath), "config.json"), nil
}

func LoadProfiles() (*ProfileStore, error) {
	path, err := ProfilesPath()
	if err != nil {
		return nil, err
	}
	store := &ProfileStore{Profiles: map[string]Profile{}, path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, store); err != nil {
		return nil, err
	}
	if store.Profiles == nil {
		store.Profiles = map[string]Profile{}
	}
	store.path = path
	return store, nil
}

func (s *ProfileStore) Save() error {
	if s.path == "" {
		path, err := ProfilesPath()
		if err != nil {
			return err
		}
		s.path = path
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}

func (s *ProfileStore) Set(name string, profile Profile) error {
	if s.Profiles == nil {
		s.Profiles = map[string]Profile{}
	}
	s.Profiles[name] = profile
	if s.Default == "" {
		s.Default = name
	}
	return s.Save()
}

func (s *ProfileStore) Delete(name string) error {
	delete(s.Profiles, name)
	if s.Default == name {
		s.Default = ""
		for k := range s.Profiles {
			s.Default = k
			break
		}
	}
	return s.Save()
}

func ResolveProfile(cfg *Config, explicit string) string {
	if explicit != "" {
		cfg.Profile = explicit
	}
	store, err := LoadProfiles()
	if err == nil {
		if cfg.Profile == "" {
			cfg.Profile = store.Default
		}
		if p, ok := store.Profiles[cfg.Profile]; ok {
			if os.Getenv("WARIWARI_API_URL") == "" && p.APIURL != "" {
				cfg.APIURL = p.APIURL
			}
			if os.Getenv("WARIWARI_GROUP") == "" && p.Group != "" {
				cfg.Group = p.Group
			}
		}
	}
	EnsureAPIURL(cfg)
	return cfg.Profile
}

func TokenKey(profile string) string {
	if profile == "" {
		return "token"
	}
	return "profile:" + profile
}

func LoadToken(profile string) (string, string) {
	if token := strings.TrimSpace(os.Getenv("WARIWARI_TOKEN")); token != "" {
		return token, "env WARIWARI_TOKEN"
	}
	if token, err := keyring.Get(serviceName, TokenKey(profile)); err == nil && token != "" {
		return token, "keyring"
	}
	if token, err := readFallbackToken(profile); err == nil && token != "" {
		return token, "file"
	}
	return "", ""
}

func SaveToken(profile, token string) (string, error) {
	if os.Getenv("WARIWARI_NO_KEYRING") == "" {
		if err := keyring.Set(serviceName, TokenKey(profile), token); err == nil {
			return "keyring", nil
		}
	}
	return "file", writeFallbackToken(profile, token)
}

func DeleteToken(profile string) {
	_ = keyring.Delete(serviceName, TokenKey(profile))
	_ = os.Remove(fallbackTokenPath(profile))
}

func credentialsDir() string {
	cfgPath, err := ConfigPath()
	if err != nil {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "wariwari", "credentials")
	}
	return filepath.Join(filepath.Dir(cfgPath), "credentials")
}

func fallbackTokenPath(profile string) string {
	name := strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(TokenKey(profile))
	return filepath.Join(credentialsDir(), name)
}

func readFallbackToken(profile string) (string, error) {
	data, err := os.ReadFile(fallbackTokenPath(profile))
	return strings.TrimSpace(string(data)), err
}

func writeFallbackToken(profile, token string) error {
	dir := credentialsDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(fallbackTokenPath(profile), []byte(token), 0o600)
}
