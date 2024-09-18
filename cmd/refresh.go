package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/vahid-haghighat/awsx/cmd/internal"
	"github.com/vahid-haghighat/awsx/utilities"
)

var refreshCmd = &cobra.Command{
	Use:               "refresh",
	Short:             "Refreshes your previously used credentials.",
	Long:              `Refreshes your previously used credentials.`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configNames := []string{"default"}
		if len(args) > 0 {
			configNames = args
		}

		configs, err := internal.ReadInternalConfig()
		if err != nil {
			if err = configCmd.RunE(cmd, args); err != nil {
				return err
			}

			configs, err = internal.ReadInternalConfig()
			if err != nil {
				return err
			}
		}

		var errs []error

		prompter := internal.Prompter{}

	Configs:
		for _, configName := range configNames {
			config, ok := configs[configName]
			if !ok {
				if err = configCmd.RunE(cmd, []string{configName}); err != nil {
					errs = append(errs, err)
					continue Configs
				}

				config = configs[configName]
			}

			var profile *internal.Profile
			if len(configs[configName].Profiles) > 1 {
				profiles := utilities.Keys(configs[configName].Profiles)
				index, _, err := prompter.Select(fmt.Sprintf("Select the profile for config \"%s\"", configName), profiles, nil)
				if err != nil {
					errs = append(errs, err)
					continue Configs
				}

				profile = configs[configName].Profiles[profiles[index]]
			} else {
				for _, p := range configs[configName].Profiles {
					profile = p
				}
			}

			if profile == nil {
				errs = append(errs, errors.New(fmt.Sprintf("no profile selected for \"%s\"", configName)))
				continue Configs
			}

			if profile.Region == "" {
				errs = append(errs, errors.New(fmt.Sprintf("no region is set for profile \"%s\" in config \"%s\"", profile.Name, configName)))
				continue Configs
			}

			oidcApi, ssoApi := internal.InitClients(configs[configName])
			err = internal.RefreshCredentials(configName, profile, oidcApi, ssoApi, config, prompter)
			if err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			message := ""
			for _, err := range errs {
				message += fmt.Sprintf("%s\n", err.Error())
			}

			return errors.New(message)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}
