package install

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"javawrapper/internal/config"
)

// InstallJDK mendownload tar.gz JDK dari URL, mengekstraknya ke installDir,
// dan mencatat instalasi di jwrapper.json.
func InstallJDK(version, url, installDir string) error {
	if version == "" {
		version = "latest"
	}

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
	var topDir string

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error baca tar: %w", err)
		}
		targetPath := filepath.Join(installDir, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if topDir == "" {
				topDir = strings.TrimSuffix(hdr.Name, "/")
			}
			if err := os.MkdirAll(targetPath, os.FileMode(hdr.Mode)); err != nil {
				return fmt.Errorf("mkdir %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("mkdir parent %s: %w", filepath.Dir(targetPath), err)
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

	// Muat config dan catat instalasi
	cfg, err := config.Load()
	if err != nil {
		cfg, err = config.LoadOrCreate()
		if err != nil {
			return fmt.Errorf("gagal muat config: %w", err)
		}
	}

	installPath := filepath.Join(installDir, topDir)

	// Cari nama distribusi dari sources
	distName := "Unknown"
	if src := cfg.GetSourceByVersion(version); src != nil {
		distName = src.Name
	}

	cfg.Installed = append(cfg.Installed, config.JDKInstalled{
		Version: version,
		Path:    installPath,
		URL:     url,
		Name:    distName,
	})
	if cfg.ActiveVersion == "" {
		cfg.ActiveVersion = version
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("gagal simpan config: %w", err)
	}

	fmt.Printf("✓ JDK %s berhasil diinstall di: %s\n", version, installPath)
	return nil
}
