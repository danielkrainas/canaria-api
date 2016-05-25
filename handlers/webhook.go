package handlers

import (
	"net/http"

	"github.com/gorilla/handlers"

	"github.com/danielkrainas/canaria-api/api/errcode"
	"github.com/danielkrainas/canaria-api/common"
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
	canary := context.GetCanary(wh)
	hookID := context.GetCanaryHookID(wh)
	updated := make([]*common.WebHook, 0)
	for _, h := range canary.Hooks {
		if h.ID != hookID {
			updated = append(updated, h)
		}
	}

	canary.Hooks = updated
	if err := getApp(wh).storage.Save(wh, canary); err != nil {
		wh.Context = context.AppendError(wh.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	context.GetLogger(wh).Print("remove hook")
	w.WriteHeader(http.StatusAccepted)
}
