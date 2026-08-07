package install

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"javawrapper/internal/config"
)

// InstallJDK mendownload tar.gz JDK dari URL, mengekstraknya ke
// <binary-dir>/versions/jdk/v<version>/, dan mencatat instalasi di jwrapper.json.
func InstallJDK(version, url, _ string) error {
	if version == "" {
		version = "latest"
	}

	// Path instalasi selalu di samping binary: ./versions/jdk/v<version>/
	installDir := config.VersionInstallDir(version)

	fmt.Printf("Mendownload JDK %s dari %s...\n", version, url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("gagal download JDK: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download gagal dengan status %s", resp.Status)
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("gagal membuat install dir: %w", err)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("bukan gzip yang valid: %w", err)
	}
	defer gz.Close()

	tarReader := tar.NewReader(gz)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error baca tar: %w", err)
		}

		// Ekstrak langsung ke installDir, strip top-level dir dari archive
		targetPath := stripTopDir(installDir, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(hdr.Mode)); err != nil {
				return fmt.Errorf("mkdir %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("mkdir parent: %w", err)
			}
			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("buat file %s: %w", targetPath, err)
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				out.Close()
				return fmt.Errorf("tulis file %s: %w", targetPath, err)
			}
			out.Close()
		}
	}

	// Catat instalasi ke config
	cfg, err := config.Load()
	if err != nil {
		cfg, err = config.LoadOrCreate()
		if err != nil {
			return fmt.Errorf("gagal muat config: %w", err)
		}
	}

	distName := "Unknown"
	if src := cfg.GetSourceByVersion(version); src != nil {
		distName = src.Name
	}

	cfg.Installed = append(cfg.Installed, config.JDKInstalled{
		Version: version,
		Path:    installDir,
		URL:     url,
		Name:    distName,
	})
	if cfg.ActiveVersion == "" {
		cfg.ActiveVersion = version
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("gagal simpan config: %w", err)
	}

	fmt.Printf("✓ JDK %s berhasil diinstall di: %s\n", version, installDir)
	return nil
}

// stripTopDir mengubah path dari archive (yang punya top-level dir) menjadi
// path di bawah installDir langsung, sehingga isi archive di-extract flat ke installDir.
// Contoh: "jbr-21.0.3/bin/java"  →  "<installDir>/bin/java"
func stripTopDir(installDir, archivePath string) string {
	// Cari separator pertama (top-level dir)
	for i, c := range archivePath {
		if c == '/' && i > 0 {
			return filepath.Join(installDir, archivePath[i:])
		}
	}
	// Tidak ada subdirektori (entry top-level dir itu sendiri)
	return installDir
}
