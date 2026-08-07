package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"javawrapper/internal/config"
)

// createTestTar membuat in-memory tar.gz: direktori testjdk/ + binary testjdk/bin/java.
func createTestTar() []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	_ = tw.WriteHeader(&tar.Header{Name: "testjdk/", Mode: 0755, Typeflag: tar.TypeDir})
	_ = tw.WriteHeader(&tar.Header{Name: "testjdk/bin/", Mode: 0755, Typeflag: tar.TypeDir})

	javaBin := []byte("#!/bin/sh\necho java mock")
	_ = tw.WriteHeader(&tar.Header{
		Name:     "testjdk/bin/java",
		Mode:     0755,
		Size:     int64(len(javaBin)),
		Typeflag: tar.TypeReg,
	})
	_, _ = tw.Write(javaBin)

	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

// setupTestConfig menulis jwrapper.json di samping test binary dan mendaftarkan cleanup-nya.
func setupTestConfig(t *testing.T, installDir string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("tidak bisa dapatkan path executable: %v", err)
	}
	cfgPath := filepath.Join(filepath.Dir(exe), "jwrapper.json")

	cfg := &config.Config{
		InstallDir:     installDir,
		DefaultVersion: "test-17",
		ActiveVersion:  "",
		Sources: []config.JDKSource{
			{
				Vendor:  "MockVendor",
				Name:    "Mock JDK",
				Version: "test-17",
				URL:     "http://mock-server/jdk.tar.gz",
				OS:      "linux",
				Arch:    "x64",
			},
		},
		Installed: []config.JDKInstalled{},
	}
	if err := config.SaveTo(cfg, cfgPath); err != nil {
		t.Fatalf("gagal buat config test: %v", err)
	}
	t.Cleanup(func() { os.Remove(cfgPath) })
	return cfgPath
}

func TestInstallJDK_MockedDownload(t *testing.T) {
	tarData := createTestTar()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(tarData)
	}))
	defer server.Close()

	installDir := t.TempDir()
	cfgPath := setupTestConfig(t, installDir)

	version := "test-17"
	if err := InstallJDK(version, server.URL, installDir); err != nil {
		t.Fatalf("InstallJDK gagal: %v", err)
	}

	// Verifikasi file terekstrak
	expectedJDKDir := filepath.Join(installDir, "testjdk")
	if _, err := os.Stat(expectedJDKDir); err != nil {
		t.Errorf("direktori JDK %s tidak ditemukan: %v", expectedJDKDir, err)
	}
	javaPath := filepath.Join(expectedJDKDir, "bin", "java")
	if _, err := os.Stat(javaPath); err != nil {
		t.Errorf("binary java %s tidak ditemukan: %v", javaPath, err)
	}

	// Verifikasi config diperbarui
	cfg, err := config.LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("gagal load config setelah install: %v", err)
	}
	found := false
	for _, v := range cfg.Installed {
		if v.Version == version && v.Path == expectedJDKDir {
			found = true
			// Pastikan nama distribusi ikut tercatat
			if v.Name != "Mock JDK" {
				t.Errorf("Installed[].Name = %s, want 'Mock JDK'", v.Name)
			}
			break
		}
	}
	if !found {
		t.Errorf("versi %s tidak tercatat di config.Installed: %+v", version, cfg.Installed)
	}
	if cfg.ActiveVersion != version {
		t.Errorf("ActiveVersion = %s, want %s", cfg.ActiveVersion, version)
	}
}

func TestInstallJDK_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	installDir := t.TempDir()
	setupTestConfig(t, installDir)

	err := InstallJDK("test", server.URL, installDir)
	if err == nil {
		t.Fatal("diharapkan error saat server return 500, tapi tidak ada error")
	}
}
