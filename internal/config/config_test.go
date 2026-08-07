package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
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
