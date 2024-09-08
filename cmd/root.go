package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/vahid-haghighat/awsx/version"
	"os"
)

var versionFlag bool

var rootCmd = &cobra.Command{
	Use:   "awsx",
	Short: "Retrieve short-living credentials via AWS SSO",
	Long:  `Retrieve short-living credentials via AWS SSO`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if versionFlag {
			fmt.Printf("awsx version: v%s\n", version.Version)
			return nil
		}
		return selectCmd.RunE(cmd, args)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Prints awsx's version")
}
