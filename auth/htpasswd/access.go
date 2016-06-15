package htpasswd

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/danielkrainas/canaria-api/auth"
	"github.com/danielkrainas/canaria-api/context"
)

type authStrategy struct {
	realm    string
	htpasswd *htpasswd
}

var _ auth.AuthStrategy = &authStrategy{}

func newAuthStrategy(options map[string]interface{}) (auth.AuthStrategy, error) {
	realm, found := options["realm"]
	if _, ok := realm.(string); !found || !ok {
		return nil, errors.New(`"realm" must be set for htpasswd auth strategy`)
	}

	path, found := options["path"]
	if _, ok := path.(string); !found || !ok {
		return nil, errors.New(`"path" must be set for the htpasswd auth strategy`)
	}

	f, err := os.Open(path.(string))
	if err != nil {
		return nil, err
	}

	defer f.Close()
	h, err := newHTPasswd(f)
	if err != nil {
		return nil, err
	}

	return &authStrategy{
		realm:    realm.(string),
		htpasswd: h,
	}, nil
}

func (as *authStrategy) Authorized(ctx context.Context, accessRecords ...auth.Access) (context.Context, error) {
	req, err := context.GetRequest(ctx)
	if err != nil {
		return nil, err
	}

	username, password, ok := req.BasicAuth()
	if !ok {
		return nil, &challenge{
			realm: as.realm,
			err:   auth.ErrAuthenticationFailure,
		}
	}

	return auth.WithUser(ctx, auth.UserInfo{Name: username}), nil
}

func (as *authStrategy) AuthenticateUser(username string, password string) error {
	return as.htpasswd.authenticateUser(username, password)
}

type challenge struct {
	realm string
	err   error
}

var _ auth.Challenge = challenge{}

func (ch challenge) SetHeaders(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", ch.realm))
}

func (ch challenge) Error() string {
	return fmt.Sprintf("basic authentication challenge for realm %q: %s", ch.realm, ch.err)
}

func init() {
	auth.Register("htpasswd", auth.StrategyFactory(newAuthStrategy))
}
