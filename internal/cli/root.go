package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"javawrapper/internal/config"
	"javawrapper/internal/env"
	"javawrapper/internal/install"
	"javawrapper/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "jwrapper",
	Short: "JavaWrapper - Go wrapper untuk mengelola JDK multi-versi",
	Long: `jwrapper membantu kamu mengelola instalasi JDK dari berbagai distribusi.
Konfigurasi disimpan di jwrapper.json di samping binary ini.`,
}

func init() {
	rootCmd.AddCommand(cmdInit())
	rootCmd.AddCommand(cmdInstall())
	rootCmd.AddCommand(cmdUse())
	rootCmd.AddCommand(cmdList())
	rootCmd.AddCommand(cmdSet())
	rootCmd.AddCommand(cmdSources())
}

// Execute menjalankan root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// ----- command definitions -----

func cmdInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Inisiasi config jwrapper.json dan folder instalasi JDK",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadOrCreate()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(cfg.InstallDir, 0755); err != nil {
				return fmt.Errorf("gagal membuat install dir: %w", err)
			}
			fmt.Printf("✓ Config disimpan di: %s\n", cfg.InstallDir)
			fmt.Printf("  Default versi  : %s\n", cfg.DefaultVersion)
			fmt.Printf("  Jumlah sources : %d\n", len(cfg.Sources))
			return cfg.Save()
		},
	}
}

func cmdInstall() *cobra.Command {
	return &cobra.Command{
		Use:   "install [versi]",
		Short: "Download dan install JDK (default: versi default dari config)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("jalankan 'jwrapper init' terlebih dahulu: %w", err)
			}
			ver := cfg.DefaultVersion
			if len(args) > 0 {
				ver = args[0]
			}
			url := cfg.GetURLForVersion(ver)
			if url == "" {
				return fmt.Errorf("tidak ada URL untuk versi %s di jwrapper.json", ver)
			}
			return install.InstallJDK(ver, url, cfg.InstallDir)
		},
	}
}

func cmdUse() *cobra.Command {
	return &cobra.Command{
		Use:   "use <versi>",
		Short: "Aktifkan versi JDK yang sudah terinstall",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("jalankan 'jwrapper init' terlebih dahulu: %w", err)
			}
			return version.Switch(cfg, args[0])
		},
	}
}

func cmdList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Tampilkan semua versi JDK yang terinstall",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("jalankan 'jwrapper init' terlebih dahulu: %w", err)
			}
			version.List(cfg)
			return nil
		},
	}
}

func cmdSet() *cobra.Command {
	return &cobra.Command{
		Use:   "set <KEY> <VALUE>",
		Short: "Set environment variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return env.Set(args[0], args[1])
		},
	}
}

func cmdSources() *cobra.Command {
	return &cobra.Command{
		Use:   "sources",
		Short: "Tampilkan daftar sumber JDK yang tersedia di config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("jalankan 'jwrapper init' terlebih dahulu: %w", err)
			}
			if len(cfg.Sources) == 0 {
				fmt.Println("Tidak ada sources di jwrapper.json.")
				return nil
			}
			fmt.Printf("%-6s %-15s %-22s %-8s %-8s %s\n", "Default", "Vendor", "Nama", "Versi", "OS", "URL")
			fmt.Println("----------------------------------------------------------------------------------------------")
			for _, s := range cfg.Sources {
				def := "  "
				if s.Version == cfg.DefaultVersion {
					def = "* "
				}
				fmt.Printf("%s     %-15s %-22s %-8s %-8s %s\n", def, s.Vendor, s.Name, s.Version, s.OS, s.URL)
			}
			return nil
		},
	}
}
