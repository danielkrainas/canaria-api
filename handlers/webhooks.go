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
	Url         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
	InsecureSSL bool   `json:"insecure_ssl"`
}

type webHookSetupRequest struct {
	Name   string              `json:"name"`
	Config *webHookSetupConfig `json:"config"`
	Events []string            `json:"events"`
	Active bool                `json:"active"`
}

func (whs *webHookSetupRequest) Create() *common.WebHook {
	wh := common.NewWebHook()
	wh.ContentType = whs.Config.ContentType
	wh.Secret = whs.Config.Secret
	wh.Url = whs.Config.Url
	wh.InsecureSSL = whs.Config.InsecureSSL
	wh.Active = whs.Active
	//wh.Name = whs.Name - not used atm
	wh.Events = make([]string, len(whs.Events))
	for i, e := range whs.Events {
		wh.Events[i] = e
	}

	return wh
}

func (wh *webhooksHandler) CreateCanaryHook(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(wh).Debug("CreateCanaryHook")
	c := context.GetCanary(wh)

	/*if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeInvalidMediaType)
		return
	}*/

	decoder := json.NewDecoder(r.Body)
	setup := &webHookSetupRequest{}
	if err := decoder.Decode(setup); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	}

	nwh := setup.Create()
	c.Hooks = append(c.Hooks, nwh)
	if err := nwh.Validate(); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	} else if err := getApp(wh).storage.Save(c); err != nil {
		wh.Context = context.AppendError(wh.Context, v1.ErrorCodeWebhookSetupInvalid.WithDetail(err))
		return
	}

	// TODO: set location header
	w.WriteHeader(http.StatusCreated)
}
