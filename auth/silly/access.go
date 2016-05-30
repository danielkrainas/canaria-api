package silly

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/danielkrainas/canaria-api/auth"
	"github.com/danielkrainas/canaria-api/context"
)

type authStrategy struct {
	realm   string
	service string
}

type challenge struct {
	realm   string
	service string
	scope   string
}

var _ auth.Challenge = challenge{}

func (ch challenge) SetHeaders(w http.ResponseWriter) {
	header := fmt.Sprintf("Bearer realm=%q,service=%q", ch.realm, ch.service)

	if ch.scope != "" {
		header = fmt.Sprintf("%s,scope=%q", header, ch.scope)
	}

	w.Header().Set("WWW-Authenticate", header)
}

func (ch challenge) Error() string {
	return fmt.Sprintf("silly authentication challenge: %#v", ch)
}

func newAuthStrategy(options map[string]interface{}) (auth.AuthStrategy, error) {
	realm, exists := options["realm"]
	if _, ok := realm.(string); !exists || !ok {
		return nil, fmt.Errorf(`"realm" must be set for silly auth strategy`)
	}

	service, exists := options["service"]
	if _, ok := service.(string); !exists || !ok {
		return nil, fmt.Errorf(`"service" must be set for silly auth strategy`)
	}

	return &authStrategy{
		realm:   realm.(string),
		service: service.(string),
	}, nil
}

func (as *authStrategy) Authorized(ctx context.Context, accessRecords ...auth.Access) (context.Context, error) {
	req, err := context.GetRequest(ctx)
	if err != nil {
		return nil, err
	}

	if req.Header.Get("Authorization") == "" {
		challenge := challenge{
			realm:   as.realm,
			service: as.service,
		}

		if len(accessRecords) > 0 {
			var scopes []string
			for _, access := range accessRecords {
				scopes = append(scopes, fmt.Sprintf("%s:%s:%s", access.Type, access.Resource, access.Action))
			}

			challenge.scope = strings.Join(scopes, " ")
		}

		return nil, &challenge
	}

	return auth.WithUser(ctx, auth.UserInfo{Name: "silly"}), nil
}

func init() {
	auth.Register("silly", auth.StrategyFactory(newAuthStrategy))
}
