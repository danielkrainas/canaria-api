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
	"github.com/satori/go.uuid"

	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/context"
)

type webhookTestHandler struct {
	context.Context
}

func webhookTestDispatcher(ctx context.Context, r *http.Request) http.Handler {
	wth := &webhookTestHandler{
		Context: ctx,
	}

	return handlers.MethodHandler{
		"GET": http.HandlerFunc(wth.PingCanaryHook),
	}
}

func (wth *webhookTestHandler) PingCanaryHook(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(wth).Debug("PingCanaryHook")
	c := context.GetCanary(wth)
	hook := context.GetCanaryHook(wth, c)

	context.GetLogger(wth).Infof("pinging hook: %s", hook.Endpoint)
	logging.Info.Printf("ping hook %s/%s", c.ID, hookId)
	err = wh.Notify(c, common.EventPing)
	if err != nil {
		// logging
		http.Error(w, fmt.Sprintf(ErrMsgBadHookf, err.Error()), http.StatusAccepted)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
