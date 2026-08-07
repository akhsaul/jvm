package config

import (
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
)

type JDKInfo struct {
    Version string `yaml:"version"`
    Path    string `yaml:"path"`
    URL     string `yaml:"url,omitempty"`
}

type Config struct {
    InstallDir        string   `yaml:"install_dir"`
    DefaultJDKURL     string   `yaml:"default_jdk_url"`
    ActiveVersion     string   `yaml:"active_version"`
    InstalledVersions []JDKInfo `yaml:"installed_versions"`
}

// defaultConfigPath returns the path to the config file located beside the executable.
func defaultConfigPath() string {
    exe, err := os.Executable()
    if err != nil {
        // fallback to current working directory
        cwd, _ := os.Getwd()
        return filepath.Join(cwd, "jwrapper.yaml")
    }
    dir := filepath.Dir(exe)
    return filepath.Join(dir, "jwrapper.yaml")
}

func Load() (*Config, error) {
    path := defaultConfigPath()
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}

func LoadOrCreate() (*Config, error) {
    cfg, err := Load()
    if err == nil {
        return cfg, nil
    }
    // create default config
    exe, _ := os.Executable()
    dir := filepath.Dir(exe)
    cfg = &Config{
        InstallDir:    filepath.Join(dir, "jdk"),
        DefaultJDKURL: "https://github.com/JetBrains/JetBrainsRuntime/releases/download/jbrsdk-17.0.9%2B0/jbrsdk-17.0.9%2B0-linux-x64.tar.gz",
        ActiveVersion: "",
        InstalledVersions: []JDKInfo{},
    }
    if err := cfg.Save(); err != nil {
        return nil, err
    }
    return cfg, nil
}

func (c *Config) Save() error {
	path := defaultConfigPath()
	return SaveTo(c, path)
}

// SaveTo menyimpan config ke path yang diberikan. Berguna untuk testing.
func SaveTo(c *Config, path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadFrom memuat config dari path yang diberikan. Berguna untuk testing.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) GetURLForVersion(v string) string {
    if v == "latest" || v == "" {
        return c.DefaultJDKURL
    }
    for _, info := range c.InstalledVersions {
        if info.Version == v && info.URL != "" {
            return info.URL
        }
    }
    return c.DefaultJDKURL
}
