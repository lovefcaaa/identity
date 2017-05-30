package main

import (
	"database/sql"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

var (
	// cfg is the global configuration for the server. It's read in at startup from
	// the config.json file and enviornment variables, see config.go for more info.
	cfg *config

	// log output
	log = logrus.New()

	// application database connection
	appDB *sql.DB

	// cookie session storage
	sessionStore *sessions.CookieStore
)

func init() {
	log.Out = os.Stderr
	log.Level = logrus.InfoLevel
	log.Formatter = &logrus.TextFormatter{
		ForceColors: true,
	}
}

func main() {
	var err error
	cfg, err = initConfig(os.Getenv("GOLANG_ENV"))
	if err != nil {
		// panic if the server is missing a vital configuration detail
		panic(fmt.Errorf("server configuration error: %s", err.Error()))
	}
	if err = initKeys(cfg); err != nil {
		panic(fmt.Errorf("server keys error: %s", err.Error()))
	}
	initOauth()

	sessionStore = sessions.NewCookieStore([]byte(cfg.SessionSecret))
	if cfg.UserCookieDomain != "" {
		// sessionStore.Options.Domain = cfg.UserCookieDomain
	}

	connectToAppDb()

	s := &http.Server{}
	// connect mux to server
	s.Handler = NewServerRoutes()

	// print notable config settings
	// printConfigInfo()

	// fire it up!
	fmt.Println("starting server on port", cfg.Port)

	// start server wrapped in a log.Fatal b/c http.ListenAndServe will not
	// return unless there's an error
	log.Fatal(StartServer(cfg, s))
}

// NewServerRoutes returns a Muxer that has all API routes.
// This makes for easy testing using httptest
func NewServerRoutes() *http.ServeMux {
	m := http.NewServeMux()

	m.HandleFunc("/.well-known/acme-challenge/", CertbotHandler)
	m.Handle("/", middleware(HealthCheckHandler))
	m.Handle("/session", middleware(SessionHandler))
	m.Handle("/logout", middleware(LogoutHandler))
	m.Handle("/session/keys", middleware(KeysHandler))
	m.Handle("/session/oauth", middleware(SessionUserTokensHandler))
	m.Handle("/session/oauth/github/repoaccess", middleware(GithubRepoAccessHandler))
	m.Handle("/jwt/publickey", middleware(JwtPublicKeyHandler))
	m.Handle("/jwt/session", middleware(JwtHandler))

	// m.Handle("/session/groups", handler)
	m.Handle("/search", middleware(UsersSearchHandler))
	m.Handle("/users", middleware(UsersHandler))
	m.Handle("/users/", middleware(UserHandler))

	m.Handle("/groups", middleware(GroupsHandler))
	m.Handle("/groups/", middleware(GroupHandler))

	m.Handle("/oauth/github", middleware(GithubOauthHandler))
	m.Handle("/oauth/github/callback", middleware(GithubOAuthCallbackHandler))

	// m.Handle("/reset", middleware(ResetPasswordHandler))
	// m.Handle("/reset/", middleware(ResetPasswordHandler))

	return m
}
