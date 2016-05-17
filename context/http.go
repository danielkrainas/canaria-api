package context

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/danielkrainas/canaria-api/uuid"
)

type httpRequestContext struct {
	Context
	startedAt time.Time
	id        string
	r         *http.Request
}

func WithRequest(ctx Context, r *http.Request) Context {
	if ctx.Value("http.request") != nil {
		panic("only one request per context")
	}

	return &httpRequestContext{
		Context:   ctx,
		startedAt: time.Now(),
		id:        uuid.Generate(),
		r:         r,
	}
}

func GetRequest(ctx Context) (*http.Request, error) {
	if r, ok := ctx.Value("http.request").(*http.Request); r != nil && ok {
		return r, nil
	}

	return nil, errors.New("no context found")
}

func GetRequestID(ctx Context) string {
	return GetStringValue(ctx, "http.request.id")
}

func (ctx *httpRequestContext) Value(key interface{}) interface{} {
	if keyStr, ok := key.(string); ok {
		if keyStr == "http.request" {
			return ctx.r
		}

		for {
			if !strings.HasPrefix(keyStr, "http.request.") {
				break
			}

			parts := strings.Split(keyStr, ".")
			if len(parts) != 3 {
				break
			}

			switch parts[2] {
			case "uri":
				return ctx.r.RequestURI

			case "method":
				return ctx.r.Method

			case "host":
				return ctx.r.Host

			case "referer":
				referer := ctx.r.Referer()
				if referer != "" {
					return referer
				}

			case "useragent":
				return ctx.r.UserAgent()

			case "id":
				return ctx.id

			case "startedat":
				return ctx.startedAt

			case "contenttype":
				contentType := ctx.r.Header.Get("Content-Type")
				if contentType != "" {
					return contentType
				}
			}

			break
		}
	}

	return ctx.Context.Value(key)
}

type muxVarsContext struct {
	Context
	vars map[string]string
}

func WithVars(ctx Context, r *http.Request) Context {
	return &muxVarsContext{
		Context: ctx,
		vars:    mux.Vars(r),
	}
}

func (ctx *muxVarsContext) Value(key interface{}) interface{} {
	if keyStr, ok := key.(string); ok {
		if keyStr == "vars" {
			return ctx.vars
		}

		if strings.HasPrefix(keyStr, "vars.") {
			keyStr = strings.TrimPrefix(keyStr, "vars.")
		}

		if value, ok := ctx.vars[keyStr]; ok {
			return value
		}
	}

	return ctx.Context.Value(key)
}

func GetRequestLogger(ctx Context) Logger {
	return GetLogger(ctx,
		"http.request.id",
		"http.request.method",
		"http.request.uri",
		"http.request.useragent")
}
