package handlers

import (
	"net/http"

	"github.com/gorilla/handlers"

	"github.com/danielkrainas/canaria-api/api/errcode"
	"github.com/danielkrainas/canaria-api/context"
)

type webhookHandler struct {
	context.Context
}

func webhookDispatcher(ctx context.Context, r *http.Request) http.Handler {
	wh := &webhookHandler{
		Context: ctx,
	}

	return handlers.MethodHandler{
		"DELETE": http.HandlerFunc(wh.RemoveCanaryHook),
	}
}

func (wh *webhookHandler) RemoveCanaryHook(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(wh).Debug("RemoveCanaryHook")
	hookID := context.GetCanaryHookID(wh)

	if err := getApp(wh).storage.Hooks().Delete(wh, hookID); err != nil {
		wh.Context = context.AppendError(wh.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	context.GetLogger(wh).Print("remove hook")
	w.WriteHeader(http.StatusAccepted)
}
