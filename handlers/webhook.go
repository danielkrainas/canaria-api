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

	"github.com/danielkrainas/canaria-api/api/v1"
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
	hookID := context.GetCanaryHookID(ctx)
	updated := make([]*common.WebHook, hookCount)
	for _, h := range c.Hooks {
		if h.ID != hookID {
			updated = append(updated, h)
		}
	}

	canary.Hooks = updated
	if err := getApp(wh).storage.Save(canary); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeInternal.WithDetail(err))
		return
	}

	context.GetLogger(wh).Print("remove hook")
	w.WriteHeader(http.StatusAccepted)
}
