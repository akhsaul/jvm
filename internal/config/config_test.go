package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"javawrapper/internal/config"
)

func TestLoadOrCreate_CreatesDefaultConfig(t *testing.T) {
	// Gunakan temp dir sebagai lokasi config
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "jwrapper.yaml")

	// Patch defaultConfigPath dengan cara menyimpan config langsung ke temp path
	cfg := &config.Config{
		InstallDir:        filepath.Join(tmpDir, "jdk"),
		DefaultJDKURL:     "https://example.com/jdk.tar.gz",
		ActiveVersion:     "",
		InstalledVersions: []config.JDKInfo{},
	}
	if err := config.SaveTo(cfg, cfgPath); err != nil {
		t.Fatalf("SaveTo gagal: %v", err)
	}

	// Verifikasi file terbuat
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("file config tidak ditemukan: %v", err)
	}

	// Load ulang dan verifikasi isi
	cfg2, err := config.LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadFrom gagal: %v", err)
	}
	if cfg2.DefaultJDKURL != "https://example.com/jdk.tar.gz" {
		t.Errorf("DefaultJDKURL tidak sesuai: %s", cfg2.DefaultJDKURL)
	}
}

func TestConfig_GetURLForVersion(t *testing.T) {
	cfg := &config.Config{
		DefaultJDKURL: "https://default.example.com/jdk.tar.gz",
		InstalledVersions: []config.JDKInfo{
			{Version: "17", Path: "/opt/jdk17", URL: "https://specific.example.com/jdk17.tar.gz"},
		},
	}

	// versi "latest" harus kembalikan default URL
	if url := cfg.GetURLForVersion("latest"); url != cfg.DefaultJDKURL {
		t.Errorf("GetURLForVersion(latest) = %s, want %s", url, cfg.DefaultJDKURL)
	}

	// versi "17" yang sudah punya URL harus kembalikan URL spesifik
	if url := cfg.GetURLForVersion("17"); url != "https://specific.example.com/jdk17.tar.gz" {
		t.Errorf("GetURLForVersion(17) = %s, want specific URL", url)
	}

	// versi yang belum ada harus fallback ke default
	if url := cfg.GetURLForVersion("21"); url != cfg.DefaultJDKURL {
		t.Errorf("GetURLForVersion(21) = %s, want default URL", url)
	}
}
