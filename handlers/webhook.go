package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/handlers"

	"github.com/danielkrainas/canaria-api/api/errcode"
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
		"PATCH":  http.HandlerFunc(wh.EditCanaryHook),
		"GET":    http.HandlerFunc(wh.GetCanaryHook),
	}
}

func (wh *webhookHandler) GetCanaryHook(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(wh).Debug("GetCanaryHook")
	hook := context.GetCanaryHook(wh)
	if err := common.ServeWebHookJSON(w, hook, http.StatusOK); err != nil {
		context.GetLogger(wh).Errorf("error sending webhook json: %v", err)
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

type editHookRequest struct {
	Config *webHookSetupConfig `json:"config, omitempty"`
	Events []string            `json:"events"`
	Active bool                `json:"active"`
}

func (edit *editHookRequest) applyTo(hook *common.WebHook) {
	if edit.Config != nil {
		edit.Config.apply(hook)
	}

	hook.Active = edit.Active
	if len(edit.Events) > 0 {
		hook.Events = make([]string, len(edit.Events))
		for i, e := range edit.Events {
			hook.Events[i] = e
		}
	}
}

func (wh *webhookHandler) EditCanaryHook(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(wh).Debug("EditCanaryHook")
	hook := context.GetCanaryHook(wh)

	decoder := json.NewDecoder(r.Body)
	edit := &editHookRequest{}
	if err := decoder.Decode(edit); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	}

	edit.applyTo(hook)
	if err := getApp(wh).storage.Hooks().Store(wh, hook); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
	}

	// TODO: location header or webhook json body
	w.WriteHeader(http.StatusOK)
}
