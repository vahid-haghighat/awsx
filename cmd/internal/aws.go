package internal

import (
	"context"
	"errors"
	"fmt"
	ssoConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	ssoTypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	oidcTypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/aws/smithy-go"

	"log"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"time"
)

const grantType = "urn:ietf:params:oauth:grant-type:device_code"
const clientType = "public"
const clientName = "awsx"

func (ati ClientInformation) IsExpired() (bool, bool) {
	return ati.AccessTokenExpiresAt.Before(time.Now()), ati.ClientSecretExpiresAt.Before(time.Now())
}

func ProcessClientInformation(configName string, startUrl string, oidcClient *ssooidc.Client) (*ClientInformation, error) {
	clientInformation, err := GetClientInformationForConfig(configName)
	if err != nil {
		return Register(configName, startUrl, oidcClient)
	}

	accessTokenExpired, clientSecretExpired := clientInformation.IsExpired()
	if clientSecretExpired {
		return Register(configName, startUrl, oidcClient)
	}
	if accessTokenExpired {
		log.Println("AccessToken expired. Start retrieving a new AccessToken.")
		clientInformation, err = HandleOutdatedAccessToken(configName, startUrl, clientInformation, oidcClient)
		if err != nil {
			return nil, err
		}
	}
	return clientInformation, nil
}

func Register(configName string, startUrl string, oidcClient *ssooidc.Client) (*ClientInformation, error) {
	clientInformation, err := registerClient(oidcClient, startUrl)
	if err != nil {
		return nil, err
	}

	clientInformation = retrieveToken(oidcClient, clientInformation)
	err = SetClientInformationForConfig(configName, clientInformation)
	if err != nil {
		return nil, err
	}

	return clientInformation, nil
}

func HandleOutdatedAccessToken(configName string, startUrl string, clientInformation *ClientInformation, oidcClient *ssooidc.Client) (*ClientInformation, error) {
	registerClientOutput := ssooidc.RegisterClientOutput{ClientId: &clientInformation.ClientId, ClientSecret: &clientInformation.ClientSecret}
	sda, err := startDeviceAuthorization(oidcClient, &registerClientOutput, startUrl)
	if err != nil {
		return nil, err
	}

	clientInformation.DeviceCode = *sda.DeviceCode

	var clientInfoPointer *ClientInformation
	clientInfoPointer = retrieveToken(oidcClient, clientInformation)
	err = SetClientInformationForConfig(configName, clientInfoPointer)
	if err != nil {
		return nil, err
	}

	return clientInfoPointer, nil
}

func generateCreateTokenInput(clientInformation *ClientInformation) ssooidc.CreateTokenInput {
	gtp := grantType
	return ssooidc.CreateTokenInput{
		ClientId:     &clientInformation.ClientId,
		ClientSecret: &clientInformation.ClientSecret,
		DeviceCode:   &clientInformation.DeviceCode,
		GrantType:    &gtp,
	}
}

func registerClient(oidc *ssooidc.Client, startUrl string) (*ClientInformation, error) {
	cn := clientName
	ct := clientType

	rci := ssooidc.RegisterClientInput{ClientName: &cn, ClientType: &ct}
	rco, err := oidc.RegisterClient(context.Background(), &rci)
	if err != nil {
		return nil, err
	}

	sdao, err := startDeviceAuthorization(oidc, rco, startUrl)
	if err != nil {
		return nil, err
	}

	return &ClientInformation{
		ClientId:                *rco.ClientId,
		ClientSecret:            *rco.ClientSecret,
		ClientSecretExpiresAt:   time.Unix(rco.ClientSecretExpiresAt, 0),
		DeviceCode:              *sdao.DeviceCode,
		VerificationUriComplete: *sdao.VerificationUriComplete,
		StartUrl:                startUrl,
	}, nil
}

func startDeviceAuthorization(ssoClient *ssooidc.Client, rco *ssooidc.RegisterClientOutput, startUrl string) (*ssooidc.StartDeviceAuthorizationOutput, error) {
	sdao, err := ssoClient.StartDeviceAuthorization(context.Background(), &ssooidc.StartDeviceAuthorizationInput{ClientId: rco.ClientId, ClientSecret: rco.ClientSecret, StartUrl: &startUrl})
	if err != nil {
		return nil, err
	}

	log.Println("Please verify your client request: " + *sdao.VerificationUriComplete)
	openUrlInBrowser(*sdao.VerificationUriComplete)
	return sdao, nil
}

func openUrlInBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("could not open %s - unsupported platform. Please open the URL manually", url)
	}
	if err != nil {
		log.Fatal(err)
	}

}

func retrieveToken(client *ssooidc.Client, info *ClientInformation) *ClientInformation {
	input := generateCreateTokenInput(info)
	var authorizationPendingException *oidcTypes.AuthorizationPendingException
	for {
		cto, err := client.CreateToken(context.Background(), &input)
		if err != nil {
			if errors.As(err, &authorizationPendingException) {
				log.Println("Still waiting for authorization...")
				time.Sleep(3 * time.Second)
				continue
			} else if smithyErr, ok := err.(*smithy.GenericAPIError); ok {
				log.Printf("API error: %v\n", smithyErr.Code)
			} else {
				log.Fatal(err)
			}
		} else {
			info.AccessToken = *cto.AccessToken
			info.AccessTokenExpiresAt = time.Now().Add(time.Hour*8 - time.Minute*5)
			return info
		}
	}
}

func InitClients(config *Config) (*ssooidc.Client, *sso.Client) {
	cfg, _ := ssoConfig.LoadDefaultConfig(context.TODO(), ssoConfig.WithRegion(config.SsoRegion))
	oidcClient := ssooidc.NewFromConfig(cfg)
	ssoClient := sso.NewFromConfig(cfg)

	return oidcClient, ssoClient
}

func RetrieveRoleInfo(accountInfo ssoTypes.AccountInfo, clientInformation *ClientInformation, ssoClient *sso.Client, selector Prompt) ssoTypes.RoleInfo {
	lari := &sso.ListAccountRolesInput{AccountId: accountInfo.AccountId, AccessToken: &clientInformation.AccessToken}
	roles, _ := ssoClient.ListAccountRoles(context.Background(), lari)

	if len(roles.RoleList) == 1 {
		log.Printf("Only one role available. Selected role: %s\n", *roles.RoleList[0].RoleName)
		return roles.RoleList[0]
	}

	sortedRoles := sortRoles(roles.RoleList)
	var rolesToSelect []string
	linePrefix := "#"

	for i, info := range sortedRoles {
		rolesToSelect = append(rolesToSelect, linePrefix+strconv.Itoa(i)+" "+*info.RoleName)
	}

	label := "Select your role - Hint: fuzzy search supported. To choose one role directly just enter #{Int}"
	indexChoice, _, _ := selector.Select(label, rolesToSelect, fuzzySearchWithPrefixAnchor(rolesToSelect, linePrefix))
	roleInfo := sortedRoles[indexChoice]
	return roleInfo
}

func RetrieveAccountInfo(clientInformation *ClientInformation, ssoClient *sso.Client, selector Prompt) ssoTypes.AccountInfo {
	var maxSize int32 = 1000 // default is 20
	lai := sso.ListAccountsInput{AccessToken: &clientInformation.AccessToken, MaxResults: &maxSize}
	accounts, _ := ssoClient.ListAccounts(context.Background(), &lai)

	sortedAccounts := sortAccounts(accounts.AccountList)

	var accountsToSelect []string
	linePrefix := "#"

	for i, info := range sortedAccounts {
		accountsToSelect = append(accountsToSelect, linePrefix+strconv.Itoa(i)+" "+*info.AccountName+" "+*info.AccountId)
	}

	label := "Select your account - Hint: fuzzy search supported. To choose one account directly just enter #{Int}"
	indexChoice, _, _ := selector.Select(label, accountsToSelect, fuzzySearchWithPrefixAnchor(accountsToSelect, linePrefix))

	accountInfo := sortedAccounts[indexChoice]

	log.Printf("Selected account: %s - %s", *accountInfo.AccountName, *accountInfo.AccountId)
	return accountInfo
}

func sortAccounts(accountList []ssoTypes.AccountInfo) []ssoTypes.AccountInfo {
	var sortedAccounts []ssoTypes.AccountInfo
	for _, info := range accountList {
		sortedAccounts = append(sortedAccounts, info)
	}
	sort.Slice(sortedAccounts, func(i, j int) bool {
		return *sortedAccounts[i].AccountName < *sortedAccounts[j].AccountName
	})
	return sortedAccounts
}

func sortRoles(rolesList []ssoTypes.RoleInfo) []ssoTypes.RoleInfo {
	var sortedRoles []ssoTypes.RoleInfo
	for _, role := range rolesList {
		sortedRoles = append(sortedRoles, role)
	}
	sort.Slice(sortedRoles, func(i, j int) bool {
		return *sortedRoles[i].RoleName < *sortedRoles[j].RoleName
	})
	return sortedRoles
}
