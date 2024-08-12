package cmd

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/spf13/cobra"
	internal2 "github.com/vahid-haghighat/awsx/cmd/internal"
	"github.com/vahid-haghighat/awsx/utilities"
	"log"
	"time"
)

var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "Lets you select a profile from available profiles on AWS SSO",
	Long:  `Lets you select a profile from available profiles on AWS SSO`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("too many config names were specified. please pass only one config name")
		}

		configs, err := internal2.ReadInternalConfig()
		if err != nil {
			if err = configCmd.RunE(cmd, args); err != nil {
				return err
			}

			configs, err = internal2.ReadInternalConfig()
			if err != nil {
				return err
			}
		}

		configName := "default"

		if len(args) == 1 {
			configName = args[0]
		}

		var profile *internal2.Profile
		if len(configs[configName].Profiles) > 1 {
			prompt := internal2.Prompter{}
			profiles := utilities.Keys(configs[configName].Profiles)
			index, _, err := prompt.Select("Select the profile", profiles, nil)
			if err != nil {
				return err
			}

			profile = configs[configName].Profiles[profiles[index]]
		} else {
			for _, p := range configs[configName].Profiles {
				profile = p
			}
		}

		if profile == nil {
			return errors.New("no profile selected")
		}

		if profile.Region == "" {
			return errors.New("no region is set for this profile")
		}

		oidcApi, ssoApi := internal2.InitClients(configs[configName])
		return start(configName, profile, oidcApi, ssoApi, configs[configName])
	},
}

func init() {
	rootCmd.AddCommand(selectCmd)
}

func start(configName string, profile *internal2.Profile, oidcClient *ssooidc.Client, ssoClient *sso.Client, config *internal2.Config) error {
	clientInformation, _ := internal2.ProcessClientInformation(configName, config.GetStartUrl(), oidcClient)

	promptSelector := internal2.Prompter{}
	accountInfo := internal2.RetrieveAccountInfo(clientInformation, ssoClient, promptSelector)
	roleInfo := internal2.RetrieveRoleInfo(accountInfo, clientInformation, ssoClient, promptSelector)
	_ = internal2.SaveUsageInformation(configName, accountInfo, roleInfo)

	rci := &sso.GetRoleCredentialsInput{AccountId: accountInfo.AccountId, RoleName: roleInfo.RoleName, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(context.Background(), rci)
	if err != nil {
		return err
	}

	err = internal2.WriteAwsConfigFile(profile.Name, config, roleCredentials.RoleCredentials)
	if err != nil {
		return err
	}

	log.Printf("Credentials expire at: %s\n", time.Unix(roleCredentials.RoleCredentials.Expiration/1000, 0))
	return nil
}
