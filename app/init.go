package app

import (
	"fmt"
	"regexp"
	"strings"
	"github.com/revel/revel"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/microsoft"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/bitbucket"
	"golang.org/x/oauth2/google"
)

var (
	// AppVersion revel app version (ldflags)
	AppVersion string

	// BuildTime revel app build-time (ldflags)
	BuildTime string
)

const (
	DEFAULT_NAMESPACE_FILTER_ACCESS = `^.*$`
	DEFAULT_NAMESPACE_FILTER_DELETE = `^.*$`
	DEFAULT_NAMESPACE_FILTER_USER = `^user-%s-`
	DEFAULT_NAMESPACE_FILTER_TEAM = `^team-%s-`
	NAMESPACE_ENVIRONMENTS = "dev,test,int,load,prod,team,user"
	NAMESPACE_TEAM   = `^[a-zA-Z0-9]{3,}$`
	NAMESPACE_APP    = `^[a-zA-Z0-9]{3,}$`
)

var (
	RegexpNamespaceEnv *regexp.Regexp
	RegexpNamespaceTeam *regexp.Regexp
	RegexpNamespaceApp *regexp.Regexp
	RegexpNamespaceFilter *regexp.Regexp
	RegexpNamespaceDeleteFilter *regexp.Regexp
	NamespaceEnvironments []string
	NamespaceFilterUser string
	NamespaceFilterTeam string

	OAuthConfig *oauth2.Config
	OAuthProvider string
)

func init() {
	// Filters is the default set of global filters.
	revel.Filters = []revel.Filter{
		revel.PanicFilter,             // Recover from panics and display an error page instead.
		revel.RouterFilter,            // Use the routing table to select the right Action
		revel.FilterConfiguringFilter, // A hook for adding or removing per-Action filters.
		revel.ParamsFilter,            // Parse parameters into Controller.Params.
		revel.SessionFilter,           // Restore and write the session cookie.
		revel.FlashFilter,             // Restore and write the flash cookie.
		revel.ValidationFilter,        // Restore kept validation errors and save new ones from cookie.
		revel.I18nFilter,              // Resolve the requested language
		HeaderFilter,                  // Add some security based headers
		revel.InterceptorFilter,       // Run interceptors around the action.
		revel.CompressFilter,          // Compress the result.
		revel.ActionInvoker,           // Invoke the action.
	}


	// Register startup functions with OnAppStart
	// revel.DevMode and revel.RunMode only work inside of OnAppStart. See Example Startup Script
	// ( order dependent )
	// revel.OnAppStart(ExampleStartupScript)
	// revel.OnAppStart(InitDB)
	// revel.OnAppStart(FillCache)
	revel.OnAppStart(InitConfig)
	revel.OnAppStart(InitOAuthConf)
}

// HeaderFilter adds common security headers
// There is a full implementation of a CSRF filter in
// https://github.com/revel/modules/tree/master/csrf
var HeaderFilter = func(c *revel.Controller, fc []revel.Filter) {
	c.Response.Out.Header().Add("X-Frame-Options", "SAMEORIGIN")
	c.Response.Out.Header().Add("X-XSS-Protection", "1; mode=block")
	c.Response.Out.Header().Add("X-Content-Type-Options", "nosniff")

	fc[0](c, fc[1:]) // Execute the next filter stage.
}

func InitConfig() {
	RegexpNamespaceFilter = regexp.MustCompile(revel.Config.StringDefault("k8s.namespace.filter.access", DEFAULT_NAMESPACE_FILTER_ACCESS))
	RegexpNamespaceDeleteFilter = regexp.MustCompile(revel.Config.StringDefault("k8s.namespace.filter.delete", DEFAULT_NAMESPACE_FILTER_DELETE))
	RegexpNamespaceTeam = regexp.MustCompile(revel.Config.StringDefault("k8s.namespace.validation.team", NAMESPACE_TEAM))
	RegexpNamespaceApp = regexp.MustCompile(revel.Config.StringDefault("k8s.namespace.validation.app", NAMESPACE_APP))
	NamespaceFilterUser = revel.Config.StringDefault("k8s.namespace.filter.user", DEFAULT_NAMESPACE_FILTER_USER)
	NamespaceFilterTeam = revel.Config.StringDefault("k8s.namespace.filter.team", DEFAULT_NAMESPACE_FILTER_TEAM)

	envList := revel.Config.StringDefault("k8s.namespace.environments", NAMESPACE_ENVIRONMENTS)
	NamespaceEnvironments = strings.Split(envList, ",")
}

func InitOAuthConf() {
	var clientId, clientSecret string
	var optExists bool
	var endpoint oauth2.Endpoint

	OAuthProvider, optExists = revel.Config.String("oauth.provider")
	if !optExists {
		panic("No oauth.provider configured")
	}

	switch OAuthProvider {
	case "google":
		endpoint = google.Endpoint
	case "github":
		endpoint = github.Endpoint
	case "live":
		endpoint = microsoft.LiveConnectEndpoint
	case "facebook":
		endpoint = facebook.Endpoint
	case "bitbucket":
		endpoint = bitbucket.Endpoint
	default:
		panic(fmt.Sprintf("oauth.provider \"%s\" is not valid", OAuthProvider))
	}

	if val, exists := revel.Config.String("oauth.endpoint.auth"); exists {
		endpoint.AuthURL = val
	}

	if val, exists := revel.Config.String("oauth.endpoint.token"); exists {
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

	OAuthConfig = &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Endpoint: endpoint,
		Scopes: []string{},
	}
}

//func ExampleStartupScript() {
//	// revel.DevMod and revel.RunMode work here
//	// Use this script to check for dev mode and set dev/prod startup scripts here!
//	if revel.DevMode == true {
//		// Dev mode
//	}
//}
