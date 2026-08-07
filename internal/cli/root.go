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
// Jika vendor (JetBrains / Zulu) SUDAH ADA di config (jwrapper.json), gunakan config saja (user override!).
// Jika vendor BELUM ADA di config, fetch otomatis dari remote (GitHub / Azul API).
func resolveEffectiveSources(cfg *config.Config, targetVendor, targetOS, targetArch string) []config.JDKSource {
	effective := append([]config.JDKSource{}, cfg.Sources...)

	// Fetch JetBrains dari GitHub jika belum di-override
	if !cfg.HasVendorSources("JetBrains") {
		if targetVendor == "" || strings.EqualFold(targetVendor, "jetbrains") {
			ghSources, err := fetcher.FetchJetBrainsSources("", cfg.LTSVersions)
			if err == nil && len(ghSources) > 0 {
				effective = append(effective, ghSources...)
			}
		}
	}

	// Fetch Zulu dari Azul API jika belum di-override
	if !cfg.HasVendorSources("Zulu") {
		if targetVendor == "" || strings.EqualFold(targetVendor, "zulu") {
			zuluSources, err := fetcher.FetchZuluSources("", cfg.LTSVersions)
			if err == nil && len(zuluSources) > 0 {
				effective = append(effective, zuluSources...)
			}
		}
	}

	// Filter berdasarkan OS dan Arch (jika ditentukan)
	if targetOS != "" || targetArch != "" {
		filtered := make([]config.JDKSource, 0, len(effective))
		for _, s := range effective {
			if s.MatchOSArch(targetOS, targetArch) {
				filtered = append(filtered, s)
			}
		}
		return filtered
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
			fmt.Printf("  OS sistem      : %s (%s)\n", config.CurrentOS(), config.CurrentArch())
			fmt.Printf("  Jumlah sources : %d\n", len(cfg.Sources))
			return cfg.Save()
		},
	}
}

func cmdInstall() *cobra.Command {
	var vendor string
	var build string
	var osFlag string
	var archFlag string
	var lts bool

	cmd := &cobra.Command{
		Use:   "install [versi]",
		Short: "Download dan install JDK",
		Long: `Download dan install JDK dari daftar sources di jwrapper.json atau provider resmi (JetBrains, Zulu).

Contoh:
  jwrapper install 25                        # match versi 25 terbaru
  jwrapper install 21 --vendor zulu          # install versi 21 dari provider Zulu
  jwrapper install 25 --vendor jetbrains     # install versi 25 dari JetBrains
  jwrapper install --lts                     # install LTS terbaru
  jwrapper install 25 --os linux --arch x64  # install spesifik OS & Arch`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("jalankan 'jwrapper init' terlebih dahulu: %w", err)
			}

			// Secara default gunakan OS dan Arch dari runtime jika flag tidak diisi
			if osFlag == "" {
				osFlag = config.CurrentOS()
			}
			if archFlag == "" {
				archFlag = config.CurrentArch()
			}

			// Ambil sources efektif dengan filter OS & Arch
			effectiveSources := resolveEffectiveSources(cfg, vendor, osFlag, archFlag)

			var src *config.JDKSource

			if lts {
				src, err = config.FindLatestLTSIn(effectiveSources, vendor)
				if err != nil {
					return err
				}
				fmt.Printf("Ditemukan LTS terbaru: %s %s (v%s build %s) untuk %s/%s\n",
					src.Vendor, src.Name, src.Version, src.Build, src.OS, src.Arch)
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

			return install.InstallJDK(src.Version, src.Build, src.URL, cfg.InstallDir)
		},
	}

	cmd.Flags().StringVar(&vendor, "vendor", "", "Filter vendor (case-insensitive), misal: jetbrains, zulu")
	cmd.Flags().StringVar(&build, "build", "", "Filter build number (case-insensitive), misal: b508.7, zulu21.34.19")
	cmd.Flags().StringVar(&osFlag, "os", "", "Filter OS (default: OS sistem saat runtime)")
	cmd.Flags().StringVar(&archFlag, "arch", "", "Filter Arsitektur (default: Arch sistem saat runtime)")
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
	var vendorFlag string
	var osFlag string
	var archFlag string
	var ltsFlag bool

	cmd := &cobra.Command{
		Use:   "sources",
		Short: "Tampilkan daftar sumber JDK yang tersedia di config/fetched",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("jalankan 'jwrapper init' terlebih dahulu: %w", err)
			}

			sources := resolveEffectiveSources(cfg, vendorFlag, osFlag, archFlag)
			if cfg.HasVendorSources("JetBrains") {
				fmt.Println("(JetBrains sources di-override dari jwrapper.json)")
			}
			if cfg.HasVendorSources("Zulu") {
				fmt.Println("(Zulu sources di-override dari jwrapper.json)")
			}

			// Filter rilis LTS jika flag --lts diaktifkan
			if ltsFlag {
				ltsSources := make([]config.JDKSource, 0, len(sources))
				for _, s := range sources {
					if s.LTS {
						ltsSources = append(ltsSources, s)
					}
				}
				sources = ltsSources
			}

			if len(sources) == 0 {
				fmt.Println("Tidak ada sources yang cocok.")
				return nil
			}

			// Urutkan sources: Vendor A-Z, Versi (desc), Build (desc)
			config.SortSources(sources)

			isDevMode := !strings.EqualFold(config.Mode, "prod")

			if isDevMode {
				fmt.Printf("%-6s %-12s %-18s %-10s %-16s %-5s %-7s %-8s %s\n", "Default", "Vendor", "Nama", "Versi", "Build", "LTS", "OS", "Arch", "URL")
				fmt.Println("--------------------------------------------------------------------------------------------------------------------------------")
			} else {
				fmt.Printf("%-6s %-12s %-18s %-10s %-16s %-5s %-7s %-8s\n", "Default", "Vendor", "Nama", "Versi", "Build", "LTS", "OS", "Arch")
				fmt.Println("----------------------------------------------------------------------------------------------------")
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
					fmt.Printf("%s     %-12s %-18s %-10s %-16s %-5s %-7s %-8s %s\n", def, s.Vendor, s.Name, s.Version, build, isLTS, s.OS, s.Arch, s.URL)
				} else {
					fmt.Printf("%s     %-12s %-18s %-10s %-16s %-5s %-7s %-8s\n", def, s.Vendor, s.Name, s.Version, build, isLTS, s.OS, s.Arch)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&vendorFlag, "vendor", "", "Filter berdasarkan Vendor (misal: jetbrains, zulu)")
	cmd.Flags().StringVar(&osFlag, "os", "", "Filter berdasarkan OS (misal: linux, darwin, windows)")
	cmd.Flags().StringVar(&archFlag, "arch", "", "Filter berdasarkan Arsitektur (misal: x64, aarch64, x86)")
	cmd.Flags().BoolVar(&ltsFlag, "lts", false, "Filter hanya menampilkan rilis LTS")
	return cmd
}


