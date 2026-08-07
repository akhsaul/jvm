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

// InstallJDK downloads a tar.gz JDK archive from the given URL, extracts it into installDir,
// and records the installation in the configuration.
func InstallJDK(version, url, installDir string) error {
    if version == "" {
        version = "latest"
    }

    // download the archive
    resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("failed to download JDK: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("download returned status %s", resp.Status)
    }

    // ensure install directory exists
    if err := os.MkdirAll(installDir, 0755); err != nil {
        return fmt.Errorf("cannot create install dir: %w", err)
    }

    // stream extraction without writing the entire archive to disk
    gz, err := gzip.NewReader(resp.Body)
    if err != nil {
        return fmt.Errorf("invalid gzip stream: %w", err)
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
            return fmt.Errorf("tar read error: %w", err)
        }
        targetPath := filepath.Join(installDir, hdr.Name)
        switch hdr.Typeflag {
        case tar.TypeDir:
            if topDir == "" {
                // first directory entry is usually the top-level folder of the archive
                topDir = hdr.Name
            }
            if err := os.MkdirAll(targetPath, os.FileMode(hdr.Mode)); err != nil {
                return fmt.Errorf("mkdir %s: %w", targetPath, err)
            }
        case tar.TypeReg:
            // ensure the parent directory exists
            if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
                return fmt.Errorf("mkdir parent %s: %w", filepath.Dir(targetPath), err)
            }
            out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
            if err != nil {
                return fmt.Errorf("open %s: %w", targetPath, err)
            }
            if _, err := io.Copy(out, tarReader); err != nil {
                out.Close()
                return fmt.Errorf("copy to %s: %w", targetPath, err)
            }
            out.Close()
        }
    }

    // Record installation in config
    cfg, err := config.Load()
    if err != nil {
        // if config missing, create default one
        cfg, err = config.LoadOrCreate()
        if err != nil {
            return fmt.Errorf("cannot load config: %w", err)
        }
    }
    // compute actual path of extracted JDK – topDir may have trailing slash
    installPath := filepath.Join(installDir, topDir)
    cfg.InstalledVersions = append(cfg.InstalledVersions, config.JDKInfo{Version: version, Path: installPath, URL: url})
    if cfg.ActiveVersion == "" {
        cfg.ActiveVersion = version
    }
    return cfg.Save()
}
