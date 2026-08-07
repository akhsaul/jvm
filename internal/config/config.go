package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// JDKSource mendefinisikan sumber download JDK yang tersedia.
type JDKSource struct {
	Vendor  string `json:"vendor"`            // nama vendor tanpa spasi, misal "JetBrains", "Eclipse"
	Name    string `json:"name"`              // nama distribusi, misal "JetBrains Runtime"
	Version string `json:"version"`           // versi JDK, misal "17", "21"
	URL     string `json:"url"`               // URL download tar.gz
	OS      string `json:"os,omitempty"`      // target OS: "linux", "windows", "darwin"
	Arch    string `json:"arch,omitempty"`    // target arch: "x64", "aarch64"
}

// Validate memastikan field Vendor tidak mengandung spasi.
func (s *JDKSource) Validate() error {
	if strings.Contains(s.Vendor, " ") {
		return fmt.Errorf("vendor %q tidak boleh mengandung spasi", s.Vendor)
	}
	if s.Vendor == "" {
		return fmt.Errorf("vendor tidak boleh kosong")
	}
	return nil
}

// JDKInstalled merepresentasikan JDK yang sudah terinstall di sistem.
type JDKInstalled struct {
	Version  string `json:"version"`           // versi JDK
	Path     string `json:"path"`              // path folder hasil ekstrak
	URL      string `json:"url,omitempty"`     // URL sumber download
	Name     string `json:"name,omitempty"`    // nama distribusi
}

// Config adalah struktur utama file jwrapper.json.
type Config struct {
	// InstallDir adalah folder tempat JDK diinstall.
	InstallDir string `json:"install_dir"`

	// DefaultVersion adalah versi JDK yang digunakan secara default saat install.
	DefaultVersion string `json:"default_version"`

	// ActiveVersion adalah versi JDK yang sedang aktif digunakan.
	ActiveVersion string `json:"active_version"`

	// Sources adalah daftar sumber JDK yang tersedia untuk di-download.
	// User bisa menambah entry baru di sini.
	Sources []JDKSource `json:"sources"`

	// Installed adalah daftar JDK yang sudah terinstall.
	Installed []JDKInstalled `json:"installed"`
}

// defaultSources mengembalikan daftar sumber JDK bawaan.
func defaultSources() []JDKSource {
	return []JDKSource{
		{
			Vendor:  "JetBrains",
			Name:    "JetBrains Runtime",
			Version: "21",
			URL:     "https://github.com/JetBrains/JetBrainsRuntime/releases/download/jbr-release-21.0.3b465.3/jbr_jcef-21.0.3-linux-x64-b465.3.tar.gz",
			OS:      "linux",
			Arch:    "x64",
		},
		{
			Vendor:  "JetBrains",
			Name:    "JetBrains Runtime",
			Version: "17",
			URL:     "https://github.com/JetBrains/JetBrainsRuntime/releases/download/jbr-release-17.0.10b1087.23/jbr_jcef-17.0.10-linux-x64-b1087.23.tar.gz",
			OS:      "linux",
			Arch:    "x64",
		},
	}
}

// defaultConfigPath mengembalikan path ke jwrapper.json di samping executable.
func defaultConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, "jwrapper.json")
	}
	return filepath.Join(filepath.Dir(exe), "jwrapper.json")
}

// VersionInstallDir mengembalikan path instalasi untuk versi tertentu:
// <dir-binary>/versions/jdk/v<version>/
func VersionInstallDir(version string) string {
	exe, err := os.Executable()
	if err != nil {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, "versions", "jdk", "v"+version)
	}
	return filepath.Join(filepath.Dir(exe), "versions", "jdk", "v"+version)
}

// Load memuat konfigurasi dari jwrapper.json di samping binary.
func Load() (*Config, error) {
	return LoadFrom(defaultConfigPath())
}

// LoadOrCreate memuat konfigurasi yang ada, atau membuat yang baru dengan nilai default.
func LoadOrCreate() (*Config, error) {
	cfg, err := Load()
	if err == nil {
		return cfg, nil
	}
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	cfg = &Config{
		InstallDir:     filepath.Join(dir, "versions", "jdk"),
		DefaultVersion: "21",
		ActiveVersion:  "",
		Sources:        defaultSources(),
		Installed:      []JDKInstalled{},
	}
	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("gagal membuat config default: %w", err)
	}
	return cfg, nil
}

// Save menyimpan konfigurasi ke jwrapper.json di samping binary.
func (c *Config) Save() error {
	return SaveTo(c, defaultConfigPath())
}

// SaveTo menyimpan config ke path yang diberikan (berguna untuk testing).
// Akan mengembalikan error jika ada JDKSource dengan vendor yang mengandung spasi.
func SaveTo(c *Config, path string) error {
	// Validasi semua sources sebelum disimpan
	for i, s := range c.Sources {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("sources[%d]: %w", i, err)
		}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("gagal marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// LoadFrom memuat config dari path yang diberikan (berguna untuk testing).
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("gagal parse jwrapper.json: %w", err)
	}
	return &cfg, nil
}

// GetURLForVersion mengembalikan URL download untuk versi tertentu.
// Prioritas: (1) versi yang sudah terinstall, (2) sources list, (3) default version dari sources.
func (c *Config) GetURLForVersion(version string) string {
	if version == "" || version == "latest" {
		version = c.DefaultVersion
	}

	// Cari di daftar sources berdasarkan versi
	for _, s := range c.Sources {
		if s.Version == version {
			return s.URL
		}
	}

	// Fallback: kembalikan URL source pertama
	if len(c.Sources) > 0 {
		return c.Sources[0].URL
	}
	return ""
}

// GetSourceByVersion mengembalikan JDKSource untuk versi tertentu, atau nil jika tidak ada.
func (c *Config) GetSourceByVersion(version string) *JDKSource {
	if version == "" || version == "latest" {
		version = c.DefaultVersion
	}
	for i := range c.Sources {
		if c.Sources[i].Version == version {
			return &c.Sources[i]
		}
	}
	return nil
}

// IsInstalled mengecek apakah versi tertentu sudah terinstall.
func (c *Config) IsInstalled(version string) bool {
	for _, v := range c.Installed {
		if v.Version == version {
			return true
		}
	}
	return false
}
