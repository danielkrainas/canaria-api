package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/danielkrainas/canaria-api/api/v1"
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
		"POST":   http.HandlerFunc(ch.PokeCanary),
		"GET":    http.HandlerFunc(ch.GetCanary),
		"HEAD":   http.HandlerFunc(ch.GetCanary),
	}
}

func (ch *canaryHandler) KillCanary(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(ch).Debug("KillCanary")
	c := context.GetCanary(ch)

	context.GetLogger(ch).Warn("killing canary")
	c.Kill()
	if err := getApp(ch).store.Save(c); err != nil {
		ch.Context = context.AppendError(ch.Context, v1.ErrorCodeInternal.WithDetail(err))
		return
	}

	// TODO: set location header for canary
	w.WriteHeader(http.StatusSeeOther)
}

func (ch *canaryHandler) PokeCanary(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(ch).Debug("PokeCanary")
	c := context.GetCanary(ch)

	context.GetLogger(ch).Info("refresh canary")
	c.Refresh()
	if err := getApp(ch).storage.Save(c); err != nil {
		ch.Context = context.AppendError(ch.Context, v1.ErrorCodeInternal.WithDetail(err))
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
	} else if err = common.ServeCanaryJSON(w, c, http.StatusOK); err != nil {
		context.GetLogger(ch).Errorf("error sending canary json: %v", err)
	}
}
