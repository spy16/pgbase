package auth

import (
	"context"
	_ "embed"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"

	"github.com/spy16/pgbase/errors"
)

const defaultSessionCookie = "_pgbase_auth"

//go:embed schema.sql
var schema string

// Init initialises auth module and returns.
func Init(conn *pgx.Conn, baseURL string, cfg Config) (*Auth, error) {
	if err := cfg.sanitise(); err != nil {
		return nil, err
	}

	cbURL := strings.TrimSuffix(baseURL, "/") + "/oauth2/cb"
	goth.UseProviders(
		google.New(cfg.Google.ClientID, cfg.Google.ClientSecret, cbURL, cfg.Google.Scopes...),
		github.New(cfg.Github.ClientID, cfg.Github.ClientSecret, cbURL, cfg.Github.Scopes...),
	)

	if _, err := conn.Exec(context.Background(), schema); err != nil {
		return nil, err
	}

	au := &Auth{
		cfg:  cfg,
		conn: conn,
	}

	return au, nil
}

// Auth represents the auth module and implements user management and
// authentication facilities.
type Auth struct {
	cfg  Config
	conn *pgx.Conn
}

type Config struct {
	SessionTTL    time.Duration `mapstructure:"session_ttl"`
	SessionCookie string        `mapstructure:"session_cookie"`
	SigningSecret string        `mapstructure:"signing_secret"`
	EnabledKinds  []string      `mapstructure:"enabled_kinds"`

	LoginPageRoute    string `mapstructure:"login_page_route"`
	RegisterPageRoute string `mapstructure:"register_page_route"`

	Google OAuthConf `mapstructure:"google"`
	Github OAuthConf `mapstructure:"github"`
}

type OAuthConf struct {
	Scopes       []string `mapstructure:"scopes"`
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
}

func (c *Config) sanitise() error {
	if c.SessionTTL <= 0 {
		c.SessionTTL = 12 * time.Hour
	}

	if c.SessionCookie == "" {
		c.SessionCookie = defaultSessionCookie
	}

	if c.SigningSecret == "" {
		return errors.InvalidInput.Hintf("signing_secret is required")
	}

	if len(c.EnabledKinds) == 0 {
		c.EnabledKinds = []string{defaultUserKind}
	}

	return nil
}
