package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Mode menyimpan mode build ("dev" atau "prod").
// Dapat diubah saat kompilasi via: -ldflags "-X javawrapper/internal/config.Mode=prod"
var Mode = "dev"

// JDKSource mendefinisikan sumber download JDK yang tersedia.
type JDKSource struct {
	Vendor  string `json:"vendor"`            // nama vendor tanpa spasi, misal "JetBrains", "Eclipse"
	Name    string `json:"name"`              // nama distribusi, misal "JetBrains Runtime"
	Version string `json:"version"`           // versi JDK, misal "25.0.4", "21", "17"
	Build   string `json:"build,omitempty"`   // nomor build, misal "b508.27", "b400"
	LTS     bool   `json:"lts"`              // true jika versi ini adalah Long-Term Support
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

// CurrentOS mengembalikan nama OS sistem saat runtime ("linux", "windows", "darwin").
func CurrentOS() string {
	return NormalizeOS(runtime.GOOS)
}

// CurrentArch mengembalikan nama arsitektur sistem saat runtime yang ternormalisasi ("x64", "aarch64", "x86").
func CurrentArch() string {
	return NormalizeArch(runtime.GOARCH)
}

// NormalizeOS merapikan nama OS ke format standar.
func NormalizeOS(osName string) string {
	osLower := strings.ToLower(osName)
	if osLower == "macos" || osLower == "osx" {
		return "darwin"
	}
	return osLower
}

// NormalizeArch merapikan nama arsitektur ke format standar ("x64", "aarch64", "x86").
func NormalizeArch(archName string) string {
	archLower := strings.ToLower(archName)
	switch archLower {
	case "amd64", "x86_64", "x64":
		return "x64"
	case "arm64", "aarch64":
		return "aarch64"
	case "386", "i386", "x86":
		return "x86"
	default:
		return archLower
	}
}

// NormalizeArchWithBitness merapikan arsitektur dengan memperhitungkan hw_bitness (misal Azul API "x86" + 64 = "x64").
func NormalizeArchWithBitness(archName string, bitness int) string {
	archLower := strings.ToLower(archName)
	if archLower == "x86" && bitness == 64 {
		return "x64"
	}
	return NormalizeArch(archName)
}

// MatchOSArch mengecek apakah source cocok dengan target OS dan Arch (case-insensitive & ternormalisasi).
func (s *JDKSource) MatchOSArch(targetOS, targetArch string) bool {
	if targetOS != "" && s.OS != "" {
		if NormalizeOS(s.OS) != NormalizeOS(targetOS) {
			return false
		}
	}
	if targetArch != "" && s.Arch != "" {
		if NormalizeArch(s.Arch) != NormalizeArch(targetArch) {
			return false
		}
	}
	return true
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

	// LTSVersions adalah daftar versi mayor yang tergolong LTS.
	LTSVersions []string `json:"lts_versions"`

	// ActiveVersion adalah versi JDK yang sedang aktif digunakan.
	ActiveVersion string `json:"active_version"`

	// Sources adalah daftar sumber JDK yang tersedia untuk di-download.
	// User bisa menambah entry baru di sini.
	Sources []JDKSource `json:"sources"`

	// Installed adalah daftar JDK yang sudah terinstall.
	Installed []JDKInstalled `json:"installed"`
}

// defaultSources mengembalikan daftar sumber JDK bawaan (kosong secara default
// agar jwrapper otomatis fetch seluruh rilis dari GitHub via Git Smart HTTP).
func defaultSources() []JDKSource {
	return []JDKSource{}
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
	cfg, err := LoadFrom(defaultConfigPath())
	if err != nil {
		return nil, err
	}
	if len(cfg.LTSVersions) == 0 {
		cfg.LTSVersions = []string{"8", "11", "17", "21"}
	}
	return cfg, nil
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
		LTSVersions:    []string{"8", "11", "17", "21"},
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
func (c *Config) GetURLForVersion(version string) string {
	if version == "" || version == "latest" {
		version = c.DefaultVersion
	}
	for _, s := range c.Sources {
		if s.Version == version {
			return s.URL
		}
	}
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

// HasVendorSources mengecek apakah ada setidaknya satu source untuk vendor tertentu di jwrapper.json (case-insensitive).
func (c *Config) HasVendorSources(vendor string) bool {
	for _, s := range c.Sources {
		if strings.EqualFold(s.Vendor, vendor) {
			return true
		}
	}
	return false
}

// FindSource mencari JDKSource berdasarkan versi dan (opsional) vendor/build dalam c.Sources.
func (c *Config) FindSource(version, vendor, build string) (*JDKSource, error) {
	return FindSourceIn(c.Sources, version, vendor, build, c.DefaultVersion)
}

// FindSourceIn mencari JDKSource dari slice sources yang diberikan.
// Logika matching:
// - versi '25' cocok dengan versi '25' atau versi minor seperti '25.0.4'
// - versi '25.0.4' cocok persis dengan '25.0.4'
// - build 'b508.7' cocok persis dengan build (case-insensitive)
func FindSourceIn(sources []JDKSource, version, vendor, build, defaultVersion string) (*JDKSource, error) {
	if version == "" || version == "latest" {
		version = defaultVersion
	}

	vendorLower := strings.ToLower(vendor)
	buildLower := strings.TrimPrefix(strings.ToLower(build), "b")

	var candidates []JDKSource
	for _, s := range sources {
		// Matching Versi:
		// 1. Persis sama (misal "25.0.4" == "25.0.4" atau "25" == "25")
		// 2. Prefix versi mayor (misal requested "25" cocok dengan "25.0.4" atau "25.0.3")
		matchVer := false
		if s.Version == version {
			matchVer = true
		} else if strings.HasPrefix(s.Version, version+".") {
			matchVer = true
		}

		if !matchVer {
			continue
		}

		// Matching Vendor:
		if vendor != "" && !strings.EqualFold(s.Vendor, vendorLower) {
			continue
		}

		// Matching Build:
		if build != "" {
			cleanSBuild := strings.TrimPrefix(strings.ToLower(s.Build), "b")
			if cleanSBuild != buildLower && !strings.EqualFold(s.Build, build) {
				continue
			}
		}

		candidates = append(candidates, s)
	}

	if len(candidates) == 0 {
		var detail []string
		if vendor != "" {
			detail = append(detail, fmt.Sprintf("vendor %q", vendor))
		}
		if build != "" {
			detail = append(detail, fmt.Sprintf("build %q", build))
		}
		detailMsg := ""
		if len(detail) > 0 {
			detailMsg = fmt.Sprintf(" dengan %s", strings.Join(detail, " dan "))
		}
		return nil, fmt.Errorf("tidak ditemukan JDK versi %s%s yang cocok di sources", version, detailMsg)
	}

	// Urutkan kandidat agar versi & build terbaru berada di posisi pertama
	SortSources(candidates)
	return &candidates[0], nil
}

// FindLatestLTS mencari source LTS dengan versi tertinggi dalam c.Sources.
func (c *Config) FindLatestLTS(vendor string) (*JDKSource, error) {
	return FindLatestLTSIn(c.Sources, vendor)
}

// FindLatestLTSIn mencari source LTS dengan versi tertinggi dari slice sources yang diberikan.
func FindLatestLTSIn(sources []JDKSource, vendor string) (*JDKSource, error) {
	var best *JDKSource
	for i := range sources {
		s := &sources[i]
		if !s.LTS {
			continue
		}
		if vendor != "" && !strings.EqualFold(s.Vendor, vendor) {
			continue
		}
		if best == nil || compareVersion(s.Version, best.Version) > 0 {
			best = s
		}
	}
	if best == nil {
		if vendor != "" {
			return nil, fmt.Errorf("tidak ada JDK LTS untuk vendor %q di sources", vendor)
		}
		return nil, fmt.Errorf("tidak ada JDK LTS di sources")
	}
	return best, nil
}


// compareVersion membandingkan dua versi numerik sederhana (misal "17", "21").
// Mengembalikan positif jika a > b, negatif jika a < b, 0 jika sama.
func compareVersion(a, b string) int {
	parseNum := func(s string) int {
		n := 0
		for _, c := range s {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			} else {
				break
			}
		}
		return n
	}
	return parseNum(a) - parseNum(b)
}

// CompareVersions membandingkan dua versi numerik seperti "25.0.4" vs "25.0.3" atau "21" vs "17".
// Return >0 jika a > b, <0 jika a < b, 0 jika sama.
func CompareVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		numA := 0
		numB := 0
		if i < len(partsA) {
			numA = parseLeadingInt(partsA[i])
		}
		if i < len(partsB) {
			numB = parseLeadingInt(partsB[i])
		}
		if numA != numB {
			return numA - numB
		}
	}
	return 0
}

func parseLeadingInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

// CompareBuilds membandingkan dua build number seperti "b508.27" vs "b400".
// Return >0 jika a > b, <0 jika a < b, 0 jika sama.
func CompareBuilds(a, b string) int {
	cleanA := strings.TrimPrefix(strings.ToLower(a), "b")
	cleanB := strings.TrimPrefix(strings.ToLower(b), "b")
	return CompareVersions(cleanA, cleanB)
}

// SortSources mengurutkan list JDKSource berdasarkan:
// 1. Vendor (ascending, A-Z)
// 2. Versi (descending, terbaru ke terlama)
// 3. Build (descending, tertinggi ke terendah)
func SortSources(sources []JDKSource) {
	sort.SliceStable(sources, func(i, j int) bool {
		// 1. Vendor (ascending, A-Z)
		vendorI := strings.ToLower(sources[i].Vendor)
		vendorJ := strings.ToLower(sources[j].Vendor)
		if vendorI != vendorJ {
			return vendorI < vendorJ
		}
		// 2. Versi (descending)
		cmpVer := CompareVersions(sources[i].Version, sources[j].Version)
		if cmpVer != 0 {
			return cmpVer > 0
		}
		// 3. Build (descending)
		cmpBuild := CompareBuilds(sources[i].Build, sources[j].Build)
		if cmpBuild != 0 {
			return cmpBuild > 0
		}
		return false
	})
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

