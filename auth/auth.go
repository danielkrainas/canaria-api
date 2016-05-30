package auth

import (
	"fmt"
	"net/http"

	"github.com/danielkrainas/canaria-api/context"
)

var (
	UserKey = "auth.user"

	UserNameKey = "auth.user.name"
)

type AuthStrategy interface {
	Authorized(ctx context.Context, access ...Access) (context.Context, error)
}

type UserInfo struct {
	Name string
}

type Challenge interface {
	error

	SetHeaders(w http.ResponseWriter)
}

type Resource struct {
	Type string
	Name string
}

type Access struct {
	Resource
	Action string
}

type StrategyFactory func(options map[string]interface{}) (AuthStrategy, error)

var strategies map[string]StrategyFactory

func init() {
	strategies = make(map[string]StrategyFactory)
}

func Register(name string, factory StrategyFactory) error {
	if _, exists := strategies[name]; exists {
		return fmt.Errorf("strategy already registered: %s", name)
	}

	strategies[name] = factory
	return nil
}

func GetStrategy(name string, options map[string]interface{}) (AuthStrategy, error) {
	if factory, exists := strategies[name]; exists {
		return factory(options)
	}

	return nil, fmt.Errorf("no authentication strategy registered with name: %s", name)
}

func WithUser(ctx context.Context, user UserInfo) context.Context {
	return userInfoContext{
		Context: ctx,
		user:    user,
	}
}

type userInfoContext struct {
	context.Context
	user UserInfo
}

func (uic userInfoContext) Value(key interface{}) interface{} {
	switch key {
	case UserKey:
		return uic.user
	case UserNameKey:
		return uic.user.Name
	}

	return uic.Context.Value(key)
}
