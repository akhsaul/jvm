package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"javawrapper/internal/config"
)

// Switch mengubah versi JDK aktif dan menulis java_home.sh di samping binary.
func Switch(cfg *config.Config, ver string) error {
	var targetPath string
	for _, v := range cfg.Installed {
		if v.Version == ver {
			targetPath = v.Path
			break
		}
	}
	if targetPath == "" {
		return fmt.Errorf("versi %s belum terinstall. gunakan 'jwrapper list' untuk melihat versi yang tersedia", ver)
	}

	cfg.ActiveVersion = ver
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("gagal menyimpan config: %w", err)
	}

	binaryDir := filepath.Dir(os.Args[0])
	wrapperPath := filepath.Join(binaryDir, "java_home.sh")
	cleanPath := strings.TrimSuffix(targetPath, "/")
	content := fmt.Sprintf("#!/bin/sh\nexport JAVA_HOME=%s\nexport PATH=$JAVA_HOME/bin:$PATH\n", cleanPath)
	if err := os.WriteFile(wrapperPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("gagal menulis java_home.sh: %w", err)
	}

	fmt.Printf("✓ Versi aktif sekarang: %s\n", ver)
	fmt.Printf("  JAVA_HOME=%s\n", cleanPath)
	fmt.Printf("  Jalankan: source %s\n", wrapperPath)
	return nil
}

// List menampilkan semua versi JDK terinstall beserta statusnya.
func List(cfg *config.Config) {
	if len(cfg.Installed) == 0 {
		fmt.Println("Belum ada JDK yang terinstall.")
		return
	}
	fmt.Printf("%-6s %-20s %-10s %s\n", "Status", "Distribusi", "Versi", "Path")
	fmt.Println(strings.Repeat("-", 70))
	for _, v := range cfg.Installed {
		active := "  "
		if v.Version == cfg.ActiveVersion {
			active = "* "
		}
		name := v.Name
		if name == "" {
			name = "Unknown"
		}
		fmt.Printf("%s     %-20s %-10s %s\n", active, name, v.Version, v.Path)
	}
}
