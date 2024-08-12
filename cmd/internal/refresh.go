package internal

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	ssoTypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"log"
	"strconv"
	"strings"
	"time"
)

func RefreshCredentials(configName string, profile *Profile, oidcClient *ssooidc.Client, ssoClient *sso.Client, config *Config, selector Prompt) error {
	clientInformation, err := GetClientInformationForConfig(configName)
	if err != nil {
		clientInformation, _ = ProcessClientInformation(configName, config.GetStartUrl(), oidcClient)
	}

	log.Printf("Using Start URL %s", clientInformation.StartUrl)

	var accountId *string
	var roleName *string

	luis, err := GetUsageInformationForConfig(configName)

	var toSelect []string
	linePrefix := "#"

	for i, info := range luis {
		if i >= config.LastUsedAccountsCount {
			break
		}
		toSelect = append(toSelect, linePrefix+strconv.Itoa(i)+" "+info.AccountName+" "+info.AccountId+" - "+info.Role)
	}

	var lui LastUsageInformation
	if len(toSelect) == 0 {
		log.Println("Nothing to refresh yet.")
		accountInfo := RetrieveAccountInfo(clientInformation, ssoClient, Prompter{})
		roleInfo := RetrieveRoleInfo(accountInfo, clientInformation, ssoClient, Prompter{})
		roleName = roleInfo.RoleName
		accountId = accountInfo.AccountId
		err := SaveUsageInformation(configName, accountInfo, roleInfo)
		if err != nil {
			return err
		}
	} else if len(toSelect) == 1 {
		log.Printf("There is only one role available for refresh")
		lui = luis[0]
	} else {
		label := "Select an account/role combination - Hint: fuzzy search supported. To choose one account directly just enter #{Int}"
		indexChoice, _, _ := selector.Select(label, toSelect, fuzzySearchWithPrefixAnchor(toSelect, linePrefix))
		lui = luis[indexChoice]
	}

	log.Printf("Attempting to refresh credentials for account [%s] with role [%s]", lui.AccountName, lui.Role)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			log.Println("Nothing to refresh yet.")
			accountInfo := RetrieveAccountInfo(clientInformation, ssoClient, Prompter{})
			roleInfo := RetrieveRoleInfo(accountInfo, clientInformation, ssoClient, Prompter{})
			roleName = roleInfo.RoleName
			accountId = accountInfo.AccountId
			err := SaveUsageInformation(configName, accountInfo, roleInfo)
			if err != nil {
				return err
			}
		}
	} else {
		accountId = &lui.AccountId
		roleName = &lui.Role
	}

	rci := &sso.GetRoleCredentialsInput{AccountId: accountId, RoleName: roleName, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(context.Background(), rci)
	if err != nil {
		return err
	}

	err = WriteAwsConfigFile(profile.Name, config, roleCredentials.RoleCredentials)
	if err != nil {
		return err
	}

	err = SetUsageInformationForConfig(configName, &lui)

	if accountId == nil || roleName == nil {
		return errors.New("no account or role found")
	}

	log.Printf("Retrieved credentials for account %s successfully", *accountId)
	log.Printf("Assumed role: %s", *roleName)
	log.Printf("Credentials expire at: %s\n", time.Unix(roleCredentials.RoleCredentials.Expiration/1000, 0))
	return nil
}

func SaveUsageInformation(configName string, accountInfo ssoTypes.AccountInfo, roleInfo ssoTypes.RoleInfo) error {
	return SetUsageInformationForConfig(configName, &LastUsageInformation{
		AccountId:   *accountInfo.AccountId,
		AccountName: *accountInfo.AccountName,
		Role:        *roleInfo.RoleName,
	})
}
