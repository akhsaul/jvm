package cli

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
    "javawrapper/internal/config"
    "javawrapper/internal/install"
    "javawrapper/internal/version"
    "javawrapper/internal/env"
)

var rootCmd = &cobra.Command{
    Use:   "jwrapper",
    Short: "JavaWrapper - simple Go wrapper for managing JDKs and env vars",
}

func init() {
    rootCmd.AddCommand(&cobra.Command{Use: "init", Short: "Initialize config and env", RunE: initCmd})
    rootCmd.AddCommand(&cobra.Command{Use: "install", Short: "Install JDK", RunE: installCmd})
    rootCmd.AddCommand(&cobra.Command{Use: "set", Short: "Set environment variable", RunE: setEnvCmd})
    rootCmd.AddCommand(&cobra.Command{Use: "use", Short: "Switch active JDK version", RunE: useCmd})
    rootCmd.AddCommand(&cobra.Command{Use: "list", Short: "List installed JDK versions", RunE: listCmd})
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func initCmd(cmd *cobra.Command, args []string) error {
    cfg, err := config.LoadOrCreate()
    if err != nil { return err }
    // ensure install dir exists
    if err := os.MkdirAll(cfg.InstallDir, 0755); err != nil { return err }
    return cfg.Save()
}

func installCmd(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil { return err }
    version := "latest"
    if len(args) > 0 { version = args[0] }
    url := cfg.GetURLForVersion(version)
    return install.InstallJDK(version, url, cfg.InstallDir)
}

func setEnvCmd(cmd *cobra.Command, args []string) error {
    if len(args) < 2 { return fmt.Errorf("usage: jwrapper set <KEY> <VALUE>") }
    return env.Set(args[0], args[1])
}

func useCmd(cmd *cobra.Command, args []string) error {
    if len(args) < 1 { return fmt.Errorf("usage: jwrapper use <VERSION>") }
    cfg, err := config.Load()
    if err != nil { return err }
    return version.Switch(cfg, args[0])
}

func listCmd(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil { return err }
    fmt.Println("Installed JDKs:")
    for _, v := range cfg.InstalledVersions {
        fmt.Printf("- %s (path: %s)\n", v.Version, v.Path)
    }
    return nil
}
