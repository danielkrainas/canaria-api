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

type webHookSetupConfig struct {
	Url         string `json:"url, omitempty"`
	ContentType string `json:"content_type, omitempty"`
	Secret      string `json:"secret, omitempty"`
	InsecureSSL bool   `json:"insecure_ssl, omitempty"`
}

func (config *webHookSetupConfig) apply(hook *common.WebHook) {
	hook.ContentType = config.ContentType
	hook.Secret = config.Secret
	hook.Url = config.Url
	hook.InsecureSSL = config.InsecureSSL
}

type webHookSetupRequest struct {
	Name   string             `json:"name"`
	Config webHookSetupConfig `json:"config"`
	Events []string           `json:"events"`
	Active bool               `json:"active"`
}

func (whs webHookSetupRequest) Create() *common.WebHook {
	hook := common.NewWebHook()
	whs.Config.apply(hook)
	hook.Active = whs.Active
	hook.Name = whs.Name
	hook.Events = make([]string, len(whs.Events))
	for i, e := range whs.Events {
		hook.Events[i] = e
	}

	return hook
}

func (wh *webhookHandler) CreateCanaryHook(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(wh).Debug("CreateCanaryHook")
	c := context.GetCanary(wh)

	decoder := json.NewDecoder(r.Body)
	setup := &webHookSetupRequest{}
	if err := decoder.Decode(setup); err != nil {
		context.GetLogger(wh).Error(err)
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	}

	nwh := setup.Create()
	nwh.CanaryID = c.ID
	if err := nwh.Validate(); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	} else if err := getApp(wh).storage.Hooks().Store(wh, nwh); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	}

	logger := context.GetLoggerWithFields(wh, map[interface{}]interface{}{
		"hook.id": nwh.ID,
	})

	logger.Printf("canary hook created for %q", nwh.Url)
	hookURL, err := getURLBuilder(wh).BuildCanaryHookURL(c.ID, nwh.ID)
	if err != nil {
		logger.Errorf("error building hook url: %v", err)
		wh.Context = context.AppendError(wh.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	w.Header().Set("X-Canary-ID", c.ID)
	w.Header().Set("X-Webhook-ID", nwh.ID)
	w.Header().Set("Location", hookURL)
	w.WriteHeader(http.StatusCreated)
}
