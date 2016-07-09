package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/handlers"

	"github.com/danielkrainas/canaria-api/actions"
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

func webhooksDispatcher(ctx context.Context, r *http.Request) http.Handler {
	wh := &webhookHandler{
		Context: ctx,
	}

	return handlers.MethodHandler{
		"PUT": http.HandlerFunc(wh.CreateCanaryHook),
	}
}

func webhookTestDispatcher(ctx context.Context, r *http.Request) http.Handler {
	wh := &webhookHandler{
		Context: ctx,
	}

	return handlers.MethodHandler{
		"GET": http.HandlerFunc(wh.PingCanaryHook),
	}
}

func (wh *webhookHandler) PingCanaryHook(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(wh).Debug("PingCanaryHook")
	c := context.GetCanary(wh)
	hook := context.GetCanaryHook(wh)

	context.GetLogger(wh).Infof("pinging hook: %s", hook.Url)
	if err := actions.Notify(wh, hook, c, common.EventPing); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookFailed.WithDetail(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func (wh *webhookHandler) EditCanaryHook(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(wh).Debug("EditCanaryHook")
	hook := context.GetCanaryHook(wh)

	updateToken := r.Header.Get(common.HeaderHookUpdateToken)
	if updateToken == "" || updateToken != hook.UpdateToken {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeUpdateTokenInvalid.WithDetail(""))
		return
	}

	decoder := json.NewDecoder(r.Body)
	edit := &common.EditHookRequest{}
	if err := decoder.Decode(edit); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	}

	hook.Update(edit, updateToken)
	if err := getApp(wh).storage.Hooks().Store(wh, hook); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
	}

	w.Header().Set(common.HeaderHookNextUpdateToken, hook.UpdateToken)
	w.Header().Set(common.HeaderCanaryID, hook.CanaryID)
	w.Header().Set(common.HeaderHookID, hook.ID)
	common.ServeWebHookJSON(w, hook, http.StatusOK)
}

func (wh *webhookHandler) CreateCanaryHook(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(wh).Debug("CreateCanaryHook")
	c := context.GetCanary(wh)

	decoder := json.NewDecoder(r.Body)
	edit := &common.EditHookRequest{}
	if err := decoder.Decode(edit); err != nil {
		context.GetLogger(wh).Error(err)
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	}

	hook := edit.Hook()
	hook.CanaryID = c.ID
	if err := hook.Validate(); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	} else if err := getApp(wh).storage.Hooks().Store(wh, hook); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	}

	logger := context.GetLoggerWithFields(wh, map[interface{}]interface{}{
		"hook.id": hook.ID,
	})

	logger.Printf("canary hook created for %q", hook.Url)
	hookURL, err := getURLBuilder(wh).BuildCanaryHookURL(c.ID, hook.ID)
	if err != nil {
		logger.Errorf("error building hook url: %v", err)
		wh.Context = context.AppendError(wh.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	w.Header().Set(common.HeaderCanaryID, c.ID)
	w.Header().Set(common.HeaderHookID, hook.ID)
	w.Header().Set("Location", hookURL)
	w.WriteHeader(http.StatusCreated)
}
