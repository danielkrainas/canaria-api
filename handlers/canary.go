package handlers

import (
	"net/http"

	"github.com/gorilla/handlers"

	"github.com/danielkrainas/canaria-api/actions"
	"github.com/danielkrainas/canaria-api/api/errcode"
	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/context"
)

type canaryHandler struct {
	context.Context
}

func canaryDispatcher(ctx context.Context, r *http.Request) http.Handler {
	ch := &canaryHandler{
		Context: ctx,
	}

	return handlers.MethodHandler{
		"DELETE": http.HandlerFunc(ch.KillCanary),
		"POST":   http.HandlerFunc(ch.UpdateCanary),
		"GET":    http.HandlerFunc(ch.GetCanary),
		"HEAD":   http.HandlerFunc(ch.GetCanary),
	}
}

func (ch *canaryHandler) KillCanary(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(ch).Debug("KillCanary")
	c := context.GetCanary(ch)

	context.GetLogger(ch).Warn("killing canary")
	c.Kill()
	if err := getApp(ch).storage.Canaries().Store(ch, c); err != nil {
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	hooks, err := getApp(ch).storage.Hooks().GetForCanary(ch, c.ID)
	if err != nil {
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	for _, wh := range hooks {
		context.GetLogger(ch).Infof("notifying %s of event %s", wh.ID, common.EventDead)
		go actions.Notify(ch, wh, c, common.EventDead)
	}

	if _, err := getApp(ch).storage.Hooks().DeleteForCanary(ch, c.ID); err != nil {
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	for _, wh := range hooks {
		context.GetLogger(ch).Infof("hook removed: %s", wh.ID)
	}

	// TODO: set location header for canary
	w.WriteHeader(http.StatusSeeOther)
}

func (ch *canaryHandler) UpdateCanary(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(ch).Debug("UpdateCanary")
	c := context.GetCanary(ch)

	context.GetLogger(ch).Info("updating canary")
	c.Refresh()
	if err := getApp(ch).storage.Canaries().Store(ch, c); err != nil {
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	// TODO: set location header for canary
	w.WriteHeader(http.StatusSeeOther)
}

func (ch *canaryHandler) GetCanary(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(ch).Debug("GetCanary")
	c := context.GetCanary(ch)
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusNoContent)
	} else if err := common.ServeCanaryJSON(w, c, http.StatusOK); err != nil {
		context.GetLogger(ch).Errorf("error sending canary json: %v", err)
	}
}
