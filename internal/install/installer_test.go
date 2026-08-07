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

// createTestTar membuat in-memory tar.gz dengan struktur:
//   testjdk/           ← top-level dir (akan di-strip)
//   testjdk/bin/
//   testjdk/bin/java
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

// setupTestConfig menulis jwrapper.json di samping test binary.
func setupTestConfig(t *testing.T, installBaseDir string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("tidak bisa dapatkan path executable: %v", err)
	}
	cfgPath := filepath.Join(filepath.Dir(exe), "jwrapper.json")

	cfg := &config.Config{
		InstallDir:     filepath.Join(installBaseDir, "versions", "jdk"),
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

	baseDir := t.TempDir()
	cfgPath := setupTestConfig(t, baseDir)

	version := "test-17"
	build := "b100"
	if err := InstallJDK(version, build, server.URL, "" /* diabaikan */); err != nil {
		t.Fatalf("InstallJDK gagal: %v", err)
	}

	// Path yang diharapkan sesuai VersionInstallDir
	expectedInstallDir := config.VersionInstallDir(version, build)

	// Verifikasi binary java ada (strip top-dir → langsung di expectedInstallDir/bin/java)
	javaPath := filepath.Join(expectedInstallDir, "bin", "java")
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
		if v.Version == version && v.Path == expectedInstallDir {
			found = true
			if v.Name != "Mock JDK" {
				t.Errorf("Installed[].Name = %s, want 'Mock JDK'", v.Name)
			}
			break
		}
	}
	if !found {
		t.Errorf("versi %s tidak tercatat di config.Installed (path=%s): %+v",
			version, expectedInstallDir, cfg.Installed)
	}
	if cfg.ActiveVersion != version {
		t.Errorf("ActiveVersion = %s, want %s", cfg.ActiveVersion, version)
	}

	// Cleanup installed dir
	t.Cleanup(func() { os.RemoveAll(expectedInstallDir) })
}

func TestInstallJDK_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	baseDir := t.TempDir()
	setupTestConfig(t, baseDir)

	err := InstallJDK("test", "", server.URL, "")
	if err == nil {
		t.Fatal("diharapkan error saat server return 500, tapi tidak ada error")
	}
}

func TestStripTopDir(t *testing.T) {
	cases := []struct {
		archivePath string
		want        string
	}{
		{"testjdk/bin/java", "/base/bin/java"},
		{"testjdk/bin/", "/base/bin"},
		{"testjdk/", "/base"},
	}
	for _, c := range cases {
		got := stripTopDir("/base", c.archivePath)
		if got != c.want {
			t.Errorf("stripTopDir(%q) = %q, want %q", c.archivePath, got, c.want)
		}
	}
}
