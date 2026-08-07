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

// createTestTar membuat in-memory tar.gz berisi direktori testjdk/ dan dummy binary java.
func createTestTar() []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	// direktori root
	_ = tw.WriteHeader(&tar.Header{
		Name:     "testjdk/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	})

	// subdirektori bin
	_ = tw.WriteHeader(&tar.Header{
		Name:     "testjdk/bin/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	})

	// dummy java binary
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

// setupTestConfig membuat config sementara di tmpDir dan mengembalikan pathnya.
// InstallJDK bergantung pada config.Load() yang membaca dari os.Executable().
// Kita patch dengan menulis config ke path yang sama dengan executable test binary.
func setupTestConfig(t *testing.T, installDir string) string {
	t.Helper()
	// path config disamping test binary
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("tidak bisa mendapatkan path executable: %v", err)
	}
	cfgPath := filepath.Join(filepath.Dir(exe), "jwrapper.yaml")

	cfg := &config.Config{
		InstallDir:        installDir,
		DefaultJDKURL:     "https://example.com/jdk.tar.gz",
		ActiveVersion:     "",
		InstalledVersions: []config.JDKInfo{},
	}
	if err := config.SaveTo(cfg, cfgPath); err != nil {
		t.Fatalf("gagal membuat config test: %v", err)
	}
	// Hapus config setelah test selesai
	t.Cleanup(func() { os.Remove(cfgPath) })
	return cfgPath
}

func TestInstallJDK_MockedDownload(t *testing.T) {
	// 1. Buat mock HTTP server yang serve tar.gz
	tarData := createTestTar()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(tarData)
	}))
	defer server.Close()

	// 2. Temp dir sebagai installDir
	installDir := t.TempDir()

	// 3. Buat config di samping executable test
	cfgPath := setupTestConfig(t, installDir)

	// 4. Jalankan installer dengan URL dari mock server
	version := "test-17"
	if err := InstallJDK(version, server.URL, installDir); err != nil {
		t.Fatalf("InstallJDK gagal: %v", err)
	}

	// 5. Verifikasi file terekstrak dengan benar
	expectedJDKDir := filepath.Join(installDir, "testjdk")
	if _, err := os.Stat(expectedJDKDir); err != nil {
		t.Errorf("direktori JDK %s tidak ditemukan: %v", expectedJDKDir, err)
	}
	javaPath := filepath.Join(expectedJDKDir, "bin", "java")
	if _, err := os.Stat(javaPath); err != nil {
		t.Errorf("binary java %s tidak ditemukan: %v", javaPath, err)
	}

	// 6. Verifikasi config diperbarui
	cfg, err := config.LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("gagal load config setelah install: %v", err)
	}
	found := false
	for _, v := range cfg.InstalledVersions {
		if v.Version == version && v.Path == expectedJDKDir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("versi %s tidak tercatat di config (InstalledVersions: %+v)", version, cfg.InstalledVersions)
	}
	if cfg.ActiveVersion != version {
		t.Errorf("ActiveVersion = %s, want %s", cfg.ActiveVersion, version)
	}
}

func TestInstallJDK_ServerError(t *testing.T) {
	// Server yang selalu return 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	installDir := t.TempDir()
	setupTestConfig(t, installDir)

	err := InstallJDK("test", server.URL, installDir)
	if err == nil {
		t.Fatal("diharapkan error ketika server return 500, tapi tidak ada error")
	}
}
