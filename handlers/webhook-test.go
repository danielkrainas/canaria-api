package handlers

import (
	"net/http"

	"github.com/danielkrainas/canaria-api/actions"
	"github.com/danielkrainas/canaria-api/api/v1"
	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/context"

	"github.com/gorilla/handlers"
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

	context.GetLogger(wth).Infof("pinging hook: %s", hook.Url)
	if err := actions.Notify(wth, hook, c, common.EventPing); err != nil {
		wth.Context = context.AppendError(wth.Context, v1.ErrorCodeWebhookFailed.WithDetail(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
