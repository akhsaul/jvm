package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"javawrapper/internal/config"
	"javawrapper/internal/env"
	"javawrapper/internal/fetcher"
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

// resolveEffectiveSources mengembalikan daftar sources yang akan digunakan.
// Jika vendor JetBrains SUDAH ADA di config (jwrapper.json), gunakan config saja (user override!).
// Jika vendor JetBrains BELUM ADA di config, fetch otomatis dari GitHub via Git Smart HTTP.
func resolveEffectiveSources(cfg *config.Config, targetVendor string) []config.JDKSource {
	// Cek apakah user sudah set vendor JetBrains di jwrapper.json
	if cfg.HasVendorSources("JetBrains") {
		// User override di jwrapper.json: abaikan fetch GitHub!
		return cfg.Sources
	}

	// Jika vendor JetBrains tidak ada di config (atau targetVendor kosong / jetbrains), fetch dari GitHub
	effective := append([]config.JDKSource{}, cfg.Sources...)
	if targetVendor == "" || strings.EqualFold(targetVendor, "jetbrains") {
		ghSources, err := fetcher.FetchJetBrainsSources("", cfg.LTSVersions)
		if err == nil && len(ghSources) > 0 {
			effective = append(effective, ghSources...)
		}
	}

	return effective
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
			fmt.Printf("  Daftar LTS     : %v\n", cfg.LTSVersions)
			fmt.Printf("  Jumlah sources : %d\n", len(cfg.Sources))
			return cfg.Save()
		},
	}
}

func cmdInstall() *cobra.Command {
	var vendor string
	var build string
	var lts bool

	cmd := &cobra.Command{
		Use:   "install [versi]",
		Short: "Download dan install JDK",
		Long: `Download dan install JDK dari daftar sources di jwrapper.json atau GitHub.

Contoh:
  jwrapper install 25                        # match versi 25 terbaru (25.0.4 b508.27)
  jwrapper install 25.0.4                    # match versi 25.0.4 terbaru
  jwrapper install 25 --build b508.7         # match versi 25 dan build b508.7
  jwrapper install 25 --vendor jetbrains     # match versi 25 dari vendor JetBrains
  jwrapper install --lts                     # install LTS terbaru`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("jalankan 'jwrapper init' terlebih dahulu: %w", err)
			}

			// Ambil sources efektif
			effectiveSources := resolveEffectiveSources(cfg, vendor)

			var src *config.JDKSource

			if lts {
				src, err = config.FindLatestLTSIn(effectiveSources, vendor)
				if err != nil {
					return err
				}
				fmt.Printf("Ditemukan LTS terbaru: %s %s (v%s build %s)\n", src.Vendor, src.Name, src.Version, src.Build)
			} else {
				ver := cfg.DefaultVersion
				if len(args) > 0 {
					ver = args[0]
				}
				src, err = config.FindSourceIn(effectiveSources, ver, vendor, build, cfg.DefaultVersion)
				if err != nil {
					return err
				}
			}

			return install.InstallJDK(src.Version, src.URL, cfg.InstallDir)
		},
	}

	cmd.Flags().StringVar(&vendor, "vendor", "", "Filter vendor (case-insensitive), misal: jetbrains, eclipse")
	cmd.Flags().StringVar(&build, "build", "", "Filter build number (case-insensitive), misal: b508.7, b508.27")
	cmd.Flags().BoolVar(&lts, "lts", false, "Install JDK LTS terbaru dari sources")
	return cmd
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
		Short: "Tampilkan daftar sumber JDK yang tersedia di config/fetched",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("jalankan 'jwrapper init' terlebih dahulu: %w", err)
			}

			sources := resolveEffectiveSources(cfg, "")
			if cfg.HasVendorSources("JetBrains") {
				fmt.Println("(JetBrains sources di-override dari jwrapper.json)")
			}

			if len(sources) == 0 {
				fmt.Println("Tidak ada sources yang tersedia.")
				return nil
			}

			// Urutkan sources: Vendor A-Z, Versi (desc), Build (desc)
			config.SortSources(sources)

			isDevMode := !strings.EqualFold(config.Mode, "prod")

			if isDevMode {
				fmt.Printf("%-6s %-15s %-22s %-10s %-12s %-6s %-8s %s\n", "Default", "Vendor", "Nama", "Versi", "Build", "LTS", "OS", "URL")
				fmt.Println("-------------------------------------------------------------------------------------------------------------------")
			} else {
				fmt.Printf("%-6s %-15s %-22s %-10s %-12s %-6s %-8s\n", "Default", "Vendor", "Nama", "Versi", "Build", "LTS", "OS")
				fmt.Println("-----------------------------------------------------------------------------------------")
			}

			for _, s := range sources {
				def := "  "
				if s.Version == cfg.DefaultVersion {
					def = "* "
				}
				isLTS := "no"
				if s.LTS {
					isLTS = "yes"
				}
				build := s.Build
				if build == "" {
					build = "-"
				}

				if isDevMode {
					fmt.Printf("%s     %-15s %-22s %-10s %-12s %-6s %-8s %s\n", def, s.Vendor, s.Name, s.Version, build, isLTS, s.OS, s.URL)
				} else {
					fmt.Printf("%s     %-15s %-22s %-10s %-12s %-6s %-8s\n", def, s.Vendor, s.Name, s.Version, build, isLTS, s.OS)
				}
			}
			return nil
		},
	}
}


