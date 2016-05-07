package silly

import (
	"fmt"

	"github.com/danielkrainas/canaria-api/auth"
)

type authStrategy struct {
	realm   string
	service string
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

func init() {
	auth.Register("silly", auth.StrategyFactory(newAuthStrategy))
}
