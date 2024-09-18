package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/vahid-haghighat/awsx/cmd/internal"
	"github.com/vahid-haghighat/awsx/utilities"
	"sort"
	"strconv"
)

var configCmd = &cobra.Command{
	Use:               "config",
	Short:             "Configures awsx",
	Long:              `Configures one or more AWS SSO configurations`,
	Example:           "awsx config my-sso-config",
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configNames := []string{"default"}
		if len(args) > 0 {
			configNames = args
		}
		return configArgs(configNames)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func configArgs(configNames []string) error {
	configs, _ := internal.ReadInternalConfig()
	if configs == nil || len(configs) == 0 {
		configs = make(map[string]*internal.Config)
	}
	prompter := internal.Prompter{}

	for _, configName := range configNames {
		if configName == "" {
			continue
		}

		config, ok := configs[configName]
		if !ok {
			config = &internal.Config{
				Id:                    "",
				SsoRegion:             "",
				Profiles:              make(map[string]*internal.Profile),
				LastUsedAccountsCount: 1,
			}
		}
		config.Complete = false

		var err error
		config.Id, err = prompter.Prompt("Start URL Id", config.Id)
		if err != nil {
			fmt.Printf("Failed to prompt for start URL Id for %s\n", configName)
			continue
		}

		config.SsoRegion, err = prompter.Prompt("SSO Region", config.SsoRegion)
		if err != nil {
			fmt.Printf("Failed to prompt for sso region for %s\n", configName)
			continue
		}

		if config.SsoRegion == "" {
			fmt.Println("SSO Region cannot be empty")
			continue
		}

		lastUsedAccountCountString, err := prompter.Prompt("Profile count to cache for refresh command", "1")
		if err != nil {
			fmt.Printf("Failed to prompt for cached profile counts for %s\n", configName)
			continue
		}

		config.LastUsedAccountsCount, err = strconv.Atoi(lastUsedAccountCountString)
		if err != nil {
			fmt.Printf("Invalid number for %s cached accounts count\n", configName)
			continue
		}

		profileNames := utilities.Keys(config.Profiles)
		sort.SliceStable(profileNames, func(i, j int) bool {
			return profileNames[i] < profileNames[j]
		})

		var profileName string
		var region string
		profilesConfigured := 0
		for {
			defaultProfileName := "default"
			if len(profileNames) > 0 {
				defaultProfileName = profileNames[0]
			}
			profileName, err = prompter.Prompt("Profile name to configure", defaultProfileName)
			if err != nil {
				fmt.Printf("Failed to prompt for %s config argument: %s\n", configName, err)
				break
			}
			if profileName == "" {
				fmt.Println("Profile name cannot be empty")
				break
			}

			if profileName == defaultProfileName && len(profileNames) > 1 {
				profileNames = profileNames[1:]
			}

			defaultRegion, found := config.Profiles[profileName]
			if !found {
				defaultRegion = &internal.Profile{
					Region: "",
				}
			}

			region, err = prompter.Prompt("Region", defaultRegion.Region)
			if err != nil {
				fmt.Printf("Failed to prompt for region for %s: %s\n", configName, err)
				break
			}
			if region == "" {
				fmt.Println("Region name cannot be empty")
				break
			}

			config.Profiles[profileName] = &internal.Profile{
				Region: region,
			}
			profilesConfigured++

			index, _, _ := prompter.Select("Do you wish to add another profile to this config?", []string{"Yes", "No"}, nil)
			if index != 0 {
				break
			}
		}

		if profilesConfigured == 0 {
			continue
		}

		config.Complete = true
		configs[configName] = config
	}

	return internal.WriteInternalConfig(configs)
}
