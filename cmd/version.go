package cmd

import (
	"fmt"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print detailed version information",
	Long:  `Display GoLogify version, Go runtime version, and platform information.`,
	Run: func(cmd *cobra.Command, args []string) {
		bold := color.New(color.Bold)
		bold.Printf("GoLogify %s\n", Version)
		fmt.Printf("  Go:       %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  Compiler: %s\n", runtime.Compiler)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
