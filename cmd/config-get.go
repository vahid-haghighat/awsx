package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/vahid-haghighat/awsx/cmd/internal"
	"gopkg.in/yaml.v3"
)

var getConfigCmd = &cobra.Command{
	Use:               "get",
	Short:             "Prints awsx's Configuration",
	Long:              `Prints awsx's Configuration`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := internal.ReadInternalConfig()
		if err != nil {
			fmt.Println("no configuration found")
			return nil
		}
		marshal, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("internal error")
		}
		fmt.Println(string(marshal))
		return nil
	},
}

func init() {
	configCmd.AddCommand(getConfigCmd)
}
