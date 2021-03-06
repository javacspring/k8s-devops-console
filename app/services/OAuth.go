package services

import (
	"fmt"
	"strings"
	"context"
	"github.com/revel/revel"
	"k8s-devops-console/app/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"github.com/coreos/go-oidc"
	githubapi "github.com/google/go-github/github"
)

var (
	OAuthProvider string
)

type OAuth struct {
	config *oauth2.Config
	oidcProvider *oidc.Provider
	provider string
}

func (o *OAuth) GetConfig() (config *oauth2.Config) {
	if o.config == nil {
		o.config = o.buildConfig()
	}
	config = o.config;
	return
}

func (o *OAuth) GetProvider() (string) {
	return o.provider
}

func (o *OAuth) AuthCodeURL(state string) (string) {
	return o.GetConfig().AuthCodeURL(state)
}

func (o *OAuth) Exchange(code string) (*oauth2.Token, error) {
	return o.GetConfig().Exchange(context.Background(), code)
}

func (o *OAuth) FetchUserInfo(token *oauth2.Token) (user models.User, error error) {
	ctx := context.Background()

	client := o.GetConfig().Client(ctx, token)

	switch strings.ToLower(o.provider) {
	case "github":
		client := githubapi.NewClient(client)
		githubUser, _, err := client.Users.Get(context.Background(), "")
		if err != nil {
			error = err
			return
		}
		user.Username = githubUser.GetLogin()
		user.Email = githubUser.GetEmail()
		user.IsAdmin = githubUser.GetSiteAdmin()
	case "azuread":
		userInfo, err := o.oidcProvider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			error = err
			return
		}

		split := strings.SplitN(userInfo.Email, "@", 1)
		user.Username = split[0]
		user.Email = userInfo.Email
	default:
		panic(fmt.Sprintf("oauth.provider \"%s\" is not valid", OAuthProvider))
	}

	return
}

func (o *OAuth) buildConfig() (config *oauth2.Config) {
	var clientId, clientSecret string
	var optExists bool
	var endpoint oauth2.Endpoint

	ctx := context.Background()

	scopes := []string{}

	o.provider, optExists = revel.Config.String("oauth.provider")
	if !optExists {
		panic("No oauth.provider configured")
	}

	switch strings.ToLower(o.provider) {
	case "github":
		endpoint = github.Endpoint
	case "azuread":
		aadTenant := "common"
		if val, exists := revel.Config.String("oauth.azuread.tenant"); exists && val != "" {
			aadTenant = val
		}

		provider, err := oidc.NewProvider(ctx, fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", aadTenant))
		if err != nil {
			panic(fmt.Sprintf("oauth.provider AzureAD init failed: %s", err))
		}

		o.oidcProvider = provider
		endpoint = provider.Endpoint()
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	default:
		panic(fmt.Sprintf("oauth.provider \"%s\" is not valid", OAuthProvider))
	}

	if val, exists := revel.Config.String("oauth.endpoint.auth"); exists && val != "" {
		endpoint.AuthURL = val
	}

	if val, exists := revel.Config.String("oauth.endpoint.token"); exists && val != "" {
		endpoint.TokenURL = val
	}

	clientId, optExists = revel.Config.String("oauth.client.id")
	if !optExists {
		panic("No oauth.client.id configured")
	}

	clientSecret, optExists = revel.Config.String("oauth.client.secret")
	if !optExists {
		panic("No oauth.client.secret configured")
	}

	config = &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Endpoint: endpoint,
		Scopes: scopes,
	}

	return
}
