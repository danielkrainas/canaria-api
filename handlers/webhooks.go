package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/handlers"

	"github.com/danielkrainas/canaria-api/api/v1"
	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/context"
)

type webhooksHandler struct {
	context.Context
}

func webhooksDispatcher(ctx context.Context, r *http.Request) http.Handler {
	wh := &webhooksHandler{
		Context: ctx,
	}

	return handlers.MethodHandler{
		"PUT": http.HandlerFunc(wh.CreateCanaryHook),
	}
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

func (wh *webhooksHandler) CreateCanaryHook(w http.ResponseWriter, r *http.Request) {
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
		context.GetLogger(wh).Debug("CreateCanaryHook2")
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	} else if err := getApp(wh).storage.Hooks().Store(wh, nwh); err != nil {
		context.GetLogger(wh).Debug("CreateCanaryHook3")
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	}

	context.GetLogger(wh).Debug("CreateCanaryHook4")

	context.GetLoggerWithFields(wh, map[interface{}]interface{}{
		"hook.id": nwh.ID,
	}).Printf("canary webhook created for %q", nwh.Url)

	// TODO: set location header
	w.WriteHeader(http.StatusCreated)
}
