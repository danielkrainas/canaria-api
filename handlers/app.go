package handlers

import (
	"fmt"
	"net/http"

	"github.com/danielkrainas/canaria-api/api/errcode"
	"github.com/danielkrainas/canaria-api/api/v1"
	"github.com/danielkrainas/canaria-api/auth"
	"github.com/danielkrainas/canaria-api/configuration"
	"github.com/danielkrainas/canaria-api/context"
	"github.com/danielkrainas/canaria-api/storage"
	"github.com/danielkrainas/canaria-api/storage/factory"

	"github.com/gorilla/mux"
)

type dispatchFunc func(ctx context.Context, r *http.Request) http.Handler

type App struct {
	context.Context

	Config *configuration.Config

	router *mux.Router

	storage storage.StorageDriver

	authStrategy auth.AuthStrategy

	readOnly bool
}

func (app *App) Value(key interface{}) interface{} {
	if ks, ok := key.(string); ok && ks == "app" {
		return app
	}

	return app.Context.Value(key)
}

type appRequestContext struct {
	context.Context
}

func getApp(ctx context.Context) *App {
	if app, err := ctx.Value("app").(*App); !err {
		return app
	}

	return nil
}

func NewApp(ctx context.Context, config *configuration.Config) *App {
	app := &App{
		Context: ctx,
		Config:  config,
		router:  v1.RouterWithPrefix(""),
	}

	app.register(v1.RouteNameBase, func(ctx context.Context, r *http.Request) http.Handler {
		return http.HandlerFunc(apiBase)
	})

	app.register(v1.RouteNameCanaries, canariesDispatcher)
	app.register(v1.RouteNameCanary, canaryDispatcher)
	app.register(v1.RouteNameWebhook, webhookDispatcher)
	app.register(v1.RouteNameWebhooks, webhooksDispatcher)
	app.register(v1.RouteNameWebhookTest, webhookTestDispatcher)

	storageParams := config.Storage.Parameters()
	if storageParams == nil {
		storageParams = make(configuration.Parameters)
	}

	storage, err := factory.Create(config.Storage.Type(), storageParams)
	if err != nil {
		panic(err)
	}

	app.storage = storage
	return app
}

func (app *App) loadWebhook(ctx *appRequestContext) error {
	canary := context.GetCanary(ctx)
	if canary != nil {
		hook := context.GetCanaryHook(ctx, canary)
		if hook == nil {
			context.GetLogger(ctx).Errorf("canary webhook does not exist: %s", context.GetCanaryHookID(ctx))
			return v1.ErrorCodeWebhookUnknown
		}

		ctx.Context = context.WithCanaryHook(ctx.Context, hook)
		ctx.Context = context.WithLogger(ctx.Context, context.GetLoggerWithField(ctx.Context, "hook.id", hook.ID))
	} else {
		context.GetLogger(ctx).Warnln("attempt to load webhook for unknown canary")
	}

	// shouldn't be in here if the canary hasn't been loaded
	return nil
}

func (app *App) loadCanary(ctx *appRequestContext) error {
	if canary, err := app.storage.Get(ctx, context.GetCanaryID(ctx)); err != nil {
		context.GetLogger(ctx).Errorf("error resolving canary: %v", err)
		// TODO: come back to this, append unknown or invalid error
		/*switch err := err.(type) {
			case
		}*/

		return v1.ErrorCodeCanaryUnknown
	} else if canary.IsDead() || canary.IsZombie() {
		if canary.IsDead() {
			context.GetLogger(ctx).Warnf("requested canary is dead: %s", canary.ID)
		} else {
			context.GetLoggerWithField(ctx, "canary.id", canary.ID).Warnf("killing zombie")
			canary.Kill()
			if err := app.storage.Save(ctx, canary); err != nil {
				context.GetLogger(ctx).Errorf("error killing zombie canary: %v", err)
			}
		}

		return v1.ErrorCodeCanaryDead
	} else {
		ctx.Context = context.WithCanary(ctx.Context, canary)
		ctx.Context = context.WithLogger(ctx.Context, context.GetLoggerWithField(ctx.Context, "canary.id", canary.ID))
	}

	return nil
}

func (app *App) dispatcher(dispatch dispatchFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := app.context(w, r)

		if err := app.authorized(w, r, ctx); err != nil {
			context.GetLogger(ctx).Warnf("error authorizing context: %v", err)
			return
		}

		ctx.Context = context.WithLogger(ctx.Context, context.GetLogger(ctx.Context, auth.UserNameKey))
		ctx.Context = context.WithErrors(ctx.Context, make(errcode.Errors, 0))

		if app.canaryIdRequired(r) {
			err := app.loadCanary(ctx)
			if err == nil && app.hookIdRequired(r) {
				err = app.loadWebhook(ctx)
			}

			if err != nil {
				ctx.Context = context.AppendError(ctx.Context, err)
				if err := errcode.ServeJSON(w, context.GetErrors(ctx)); err != nil {
					context.GetLogger(ctx).Errorf("error serving error json: %v (from %v)", err, context.GetErrors(ctx))
				}
			}
		}

		dispatch(ctx, r).ServeHTTP(w, r)

		if errors := context.GetErrors(ctx); errors.Len() > 0 {
			if err := errcode.ServeJSON(w, errors); err != nil {
				context.GetLogger(ctx).Errorf("error serving error json: %v (from %s)", err, errors)
			}

			app.logError(ctx, errors)
		}
	})
}

func (app *App) logError(ctx context.Context, errors errcode.Errors) {
	for _, err := range errors {
		var lctx context.Context

		switch err.(type) {
		case errcode.Error:
			e, _ := err.(errcode.Error)
			lctx = context.WithValue(ctx, "err.code", e.Code)
			lctx = context.WithValue(lctx, "err.message", e.Code.Message())
			lctx = context.WithValue(lctx, "err.detail", e.Detail)
		case errcode.ErrorCode:
			e, _ := err.(errcode.ErrorCode)
			lctx = context.WithValue(ctx, "err.code", e)
			lctx = context.WithValue(lctx, "err.message", e.Message())
		default:
			// normal "error"
			lctx = context.WithValue(ctx, "err.code", errcode.ErrorCodeUnknown)
			lctx = context.WithValue(lctx, "err.message", err.Error())
		}

		lctx = context.WithLogger(ctx, context.GetLogger(lctx,
			"err.code",
			"err.message",
			"err.detail"))

		context.GetResponseLogger(lctx).Errorf("response completed with error")
	}
}

func (app *App) authorized(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
	context.GetLogger(ctx).Debug("authorizing request")

	if app.authStrategy == nil {
		return nil
	}

	// TODO: actually write this
	return nil
}

func (app *App) context(w http.ResponseWriter, r *http.Request) *appRequestContext {
	ctx := context.DefaultContextManager.Context(app, w, r)
	ctx = context.WithVars(ctx, r)
	ctx = context.WithLogger(ctx, context.GetLogger(ctx, "vars.id"))
	return &appRequestContext{
		Context: ctx,
	}
}

func (app *App) register(routeName string, dispatch dispatchFunc) {
	app.router.GetRoute(routeName).Handler(app.dispatcher(dispatch))
}

func (app *App) canaryIdRequired(r *http.Request) bool {
	route := mux.CurrentRoute(r)
	routeName := route.GetName()
	return route == nil || routeName != v1.RouteNameBase
}

func (app *App) hookIdRequired(r *http.Request) bool {
	route := mux.CurrentRoute(r)
	routeName := route.GetName()
	return route == nil || routeName == v1.RouteNameWebhook || routeName == v1.RouteNameWebhookTest
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := context.DefaultContextManager.Context(app, w, r)
	defer context.DefaultContextManager.Release(ctx)
	defer func() {
		status, ok := ctx.Value("http.response.status").(int)
		if ok && status >= 200 && status <= 399 {
			context.GetResponseLogger(ctx).Infof("response completed")
		}
	}()

	var err error
	w, err = context.GetResponseWriter(ctx)
	if err != nil {
		context.GetLogger(ctx).Warnf("response writer not found in context")
	}

	w.Header().Add("X-CANARIA-VERSION", context.GetVersion(ctx))
	app.router.ServeHTTP(w, r)
}

func apiBase(w http.ResponseWriter, r *http.Request) {
	const emptyJSON = "{}"

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprint(len(emptyJSON)))
	fmt.Fprint(w, emptyJSON)
}
