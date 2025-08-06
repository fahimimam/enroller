package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var cfgPath string

// rootCmd is the root of all sub commands in the binary
// it doesn't have a Run method as it executes other sub commands
//var rootCmd = &cobra.Command{
//	Use:     "auth",
//	Short:   "auth is a http server to serve authentication",
//	Version: "1.0.0",
//}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	r := newRootCmd()
	if err := r.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	// initConfig reads in config file and ENV variables if set.
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:   "Triapex",
		Short: "A Member for the RESTful APIs",
		Long:  `A Member for the RESTful APIs`,
	}

	rootCmd.AddCommand(
		srvCmd,
		migrationRoot,
	)

	return rootCmd
}
