package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"javawrapper/internal/config"
)

func TestSaveAndLoadFrom_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "jwrapper.json")

	cfg := &config.Config{
		InstallDir:     filepath.Join(tmpDir, "jdk"),
		DefaultVersion: "21",
		ActiveVersion:  "",
		Sources: []config.JDKSource{
			{
				Vendor:  "JetBrains",
				Name:    "JetBrains Runtime",
				Version: "21",
				URL:     "https://example.com/jbr21.tar.gz",
				OS:      "linux",
				Arch:    "x64",
			},
		},
		Installed: []config.JDKInstalled{},
	}

	// Simpan ke JSON
	if err := config.SaveTo(cfg, cfgPath); err != nil {
		t.Fatalf("SaveTo gagal: %v", err)
	}

	// Pastikan file terbuat dan valid JSON
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("file config tidak ditemukan: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("file bukan JSON valid: %v", err)
	}

	// Load ulang dan verifikasi isi
	cfg2, err := config.LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadFrom gagal: %v", err)
	}
	if cfg2.DefaultVersion != "21" {
		t.Errorf("DefaultVersion = %s, want 21", cfg2.DefaultVersion)
	}
	if len(cfg2.Sources) != 1 {
		t.Errorf("Sources len = %d, want 1", len(cfg2.Sources))
	}
	if cfg2.Sources[0].URL != "https://example.com/jbr21.tar.gz" {
		t.Errorf("Sources[0].URL = %s, want https://example.com/jbr21.tar.gz", cfg2.Sources[0].URL)
	}
}

func TestConfig_GetURLForVersion(t *testing.T) {
	cfg := &config.Config{
		DefaultVersion: "21",
		Sources: []config.JDKSource{
			{Vendor: "Oracle", Name: "OpenJDK", Version: "17", URL: "https://example.com/jbr17.tar.gz", OS: "linux"},
			{Vendor: "JetBrains", Name: "JBR", Version: "21", URL: "https://example.com/jbr21.tar.gz", OS: "linux"},
		},
	}

	// "latest" atau "" harus kembalikan URL dari DefaultVersion
	if url := cfg.GetURLForVersion("latest"); url != "https://example.com/jbr21.tar.gz" {
		t.Errorf("GetURLForVersion(latest) = %s, want jbr21 URL", url)
	}
	if url := cfg.GetURLForVersion(""); url != "https://example.com/jbr21.tar.gz" {
		t.Errorf("GetURLForVersion('') = %s, want jbr21 URL", url)
	}

	// Versi spesifik
	if url := cfg.GetURLForVersion("17"); url != "https://example.com/jbr17.tar.gz" {
		t.Errorf("GetURLForVersion(17) = %s, want jbr17 URL", url)
	}

	// Versi yang tidak ada → fallback ke sources[0]
	if url := cfg.GetURLForVersion("99"); url != "https://example.com/jbr17.tar.gz" {
		t.Errorf("GetURLForVersion(99) = %s, want fallback to sources[0]", url)
	}
}

func TestConfig_GetSourceByVersion(t *testing.T) {
	cfg := &config.Config{
		DefaultVersion: "21",
		Sources: []config.JDKSource{
			{Vendor: "JetBrains", Name: "JBR", Version: "21", URL: "https://example.com/jbr21.tar.gz"},
		},
	}

	src := cfg.GetSourceByVersion("21")
	if src == nil {
		t.Fatal("GetSourceByVersion(21) = nil, want non-nil")
	}
	if src.Name != "JBR" {
		t.Errorf("src.Name = %s, want JBR", src.Name)
	}

	if src2 := cfg.GetSourceByVersion("99"); src2 != nil {
		t.Errorf("GetSourceByVersion(99) = non-nil, want nil")
	}
}

func TestConfig_IsInstalled(t *testing.T) {
	cfg := &config.Config{
		Installed: []config.JDKInstalled{
			{Version: "17", Path: "/opt/jdk17"},
		},
	}
	if !cfg.IsInstalled("17") {
		t.Error("IsInstalled(17) = false, want true")
	}
	if cfg.IsInstalled("21") {
		t.Error("IsInstalled(21) = true, want false")
	}
}

func TestVendorValidation(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := tmpDir + "/jwrapper.json"

	// Vendor dengan spasi harus gagal
	cfgBad := &config.Config{
		InstallDir:     tmpDir,
		DefaultVersion: "21",
		Sources: []config.JDKSource{
			{Vendor: "Jet Brains", Name: "JBR", Version: "21", URL: "https://example.com/jbr21.tar.gz"},
		},
		Installed: []config.JDKInstalled{},
	}
	if err := config.SaveTo(cfgBad, cfgPath); err == nil {
		t.Error("SaveTo dengan vendor spasi harus mengembalikan error, tapi tidak ada error")
	}

	// Vendor tanpa spasi harus sukses
	cfgGood := &config.Config{
		InstallDir:     tmpDir,
		DefaultVersion: "21",
		Sources: []config.JDKSource{
			{Vendor: "JetBrains", Name: "JBR", Version: "21", URL: "https://example.com/jbr21.tar.gz"},
		},
		Installed: []config.JDKInstalled{},
	}
	if err := config.SaveTo(cfgGood, cfgPath); err != nil {
		t.Errorf("SaveTo dengan vendor valid gagal: %v", err)
	}

	// Vendor kosong harus gagal
	cfgEmpty := &config.Config{
		InstallDir:     tmpDir,
		DefaultVersion: "21",
		Sources: []config.JDKSource{
			{Vendor: "", Name: "JBR", Version: "21", URL: "https://example.com/jbr21.tar.gz"},
		},
		Installed: []config.JDKInstalled{},
	}
	if err := config.SaveTo(cfgEmpty, cfgPath); err == nil {
		t.Error("SaveTo dengan vendor kosong harus mengembalikan error, tapi tidak ada error")
	}
}

func testSources() []config.JDKSource {
	return []config.JDKSource{
		{Vendor: "JetBrains", Name: "JBR", Version: "17.0.10", Build: "b1087.23", LTS: true, URL: "https://example.com/jbr17.tar.gz"},
		{Vendor: "JetBrains", Name: "JBR", Version: "21.0.3", Build: "b465.3", LTS: true, URL: "https://example.com/jbr21.tar.gz"},
		{Vendor: "JetBrains", Name: "JBR", Version: "25.0.4", Build: "b508.27", LTS: false, URL: "https://example.com/jbr25.tar.gz"},
		{Vendor: "JetBrains", Name: "JBR", Version: "25.0.4", Build: "b400", LTS: false, URL: "https://example.com/jbr25b400.tar.gz"},
		{Vendor: "Eclipse", Name: "Temurin", Version: "21", Build: "b1", LTS: true, URL: "https://example.com/temurin21.tar.gz"},
	}
}

func TestFindSource(t *testing.T) {
	cfg := &config.Config{DefaultVersion: "21", Sources: testSources()}

	// 1. Match versi '21' dengan vendor 'jetbrains' → 21.0.3 (JBR)
	src, err := cfg.FindSource("21", "jetbrains", "")
	if err != nil {
		t.Fatalf("FindSource(21, jetbrains, '') error: %v", err)
	}
	if src.Version != "21.0.3" {
		t.Errorf("src.Version = %s, want 21.0.3", src.Version)
	}

	// 2. Major '25' cocok dengan '25.0.4' (build tertinggi b508.27)
	src, err = cfg.FindSource("25", "jetbrains", "")
	if err != nil {
		t.Fatalf("FindSource(25, jetbrains, '') error: %v", err)
	}
	if src.Version != "25.0.4" || src.Build != "b508.27" {
		t.Errorf("src = %+v, want 25.0.4 b508.27", src)
	}

	// 3. Match spesifik build 'b400'
	src, err = cfg.FindSource("25", "jetbrains", "b400")
	if err != nil {
		t.Fatalf("FindSource(25, jetbrains, b400) error: %v", err)
	}
	if src.Build != "b400" {
		t.Errorf("src.Build = %s, want b400", src.Build)
	}

	// 4. Build tidak ada → return error
	_, err = cfg.FindSource("25", "", "b508.7")
	if err == nil {
		t.Error("FindSource dengan build non-existent harus error")
	}

	// 5. Versi tidak ada → error
	_, err = cfg.FindSource("99", "", "")
	if err == nil {
		t.Error("FindSource(99, '', '') harus error")
	}
}

func TestFindLatestLTS(t *testing.T) {
	cfg := &config.Config{DefaultVersion: "21", Sources: testSources()}

	// Tanpa filter vendor → LTS tertinggi adalah v21.0.3
	src, err := cfg.FindLatestLTS("")
	if err != nil {
		t.Fatalf("FindLatestLTS('') error: %v", err)
	}
	if src.Version != "21.0.3" {
		t.Errorf("FindLatestLTS('') Version = %s, want 21.0.3", src.Version)
	}

	// Filter vendor JetBrains (case-insensitive) → LTS tertinggi dari JetBrains adalah v21.0.3
	src, err = cfg.FindLatestLTS("jetbrains")
	if err != nil {
		t.Fatalf("FindLatestLTS('jetbrains') error: %v", err)
	}
	if src.Version != "21.0.3" || src.Vendor != "JetBrains" {
		t.Errorf("FindLatestLTS('jetbrains') = %+v, want JetBrains v21.0.3", src)
	}

	// v25.0.4 tidak LTS → tidak boleh terpilih walaupun versi tertinggi
	for _, s := range cfg.Sources {
		if strings.HasPrefix(s.Version, "25") && s.LTS {
			t.Error("v25 seharusnya tidak LTS di testSources")
		}
	}

	// Vendor yang tidak ada → error
	_, err = cfg.FindLatestLTS("Oracle")
	if err == nil {
		t.Error("FindLatestLTS('Oracle') harus error, tapi tidak ada error")
	}
}

func TestSortSources(t *testing.T) {
	sources := []config.JDKSource{
		{Vendor: "Zulu", Version: "25", Build: "b300"},
		{Vendor: "JetBrains", Version: "17", Build: "b100"},
		{Vendor: "JetBrains", Version: "25.0.4", Build: "b400"},
		{Vendor: "JetBrains", Version: "25.0.4", Build: "b508.27"},
	}

	config.SortSources(sources)

	// Urutan yang diharapkan:
	// 0: JetBrains 25.0.4 b508.27
	// 1: JetBrains 25.0.4 b400
	// 2: JetBrains 17 b100
	// 3: Zulu 25 b300

	if sources[0].Vendor != "JetBrains" || sources[0].Version != "25.0.4" || sources[0].Build != "b508.27" {
		t.Errorf("sources[0] unexpected: %+v", sources[0])
	}
	if sources[1].Vendor != "JetBrains" || sources[1].Version != "25.0.4" || sources[1].Build != "b400" {
		t.Errorf("sources[1] unexpected: %+v", sources[1])
	}
	if sources[2].Vendor != "JetBrains" || sources[2].Version != "17" || sources[2].Build != "b100" {
		t.Errorf("sources[2] unexpected: %+v", sources[2])
	}
	if sources[3].Vendor != "Zulu" || sources[3].Version != "25" || sources[3].Build != "b300" {
		t.Errorf("sources[3] unexpected: %+v", sources[3])
	}
}


