package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/danielkrainas/canaria-api/api/errcode"
	"github.com/danielkrainas/canaria-api/api/v1"
	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/context"
	"github.com/danielkrainas/canaria-api/uuid"

	"github.com/gorilla/handlers"
)

type canariesHandler struct {
	context.Context
}

func canariesDispatcher(ctx context.Context, r *http.Request) http.Handler {
	ch := &canariesHandler{
		Context: ctx,
	}

	return handlers.MethodHandler{
		"PUT": http.HandlerFunc(ch.StoreCanary),
	}
}

type canaryRequest struct {
	TimeToLive int64    `json:"ttl"`
	Title      string   `json:"title"`
	Message    string   `json:"message"`
	Signature  string   `json:"signature"`
	Tags       []string `json:"tags"`
}

func (r *canaryRequest) Canary() *common.Canary {
	d := &common.Canary{
		ID:         uuid.Generate(),
		TimeToLive: r.TimeToLive,
		Title:      r.Title,
		Message:    r.Message,
		UpdatedAt:  0,
		Tags:       r.Tags,
		Hooks:      []*common.WebHook{},
		Signature:  r.Signature,
	}

	d.Refresh()
	return d
}

func (ch *canariesHandler) StoreCanary(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(ch).Debug("StoreCanary")

	decoder := json.NewDecoder(r.Body)
	cr := &canaryRequest{}
	if err := decoder.Decode(cr); err != nil {
		ch.Context = context.AppendError(ch.Context, v1.ErrorCodeCanaryInvalid.WithDetail(err))
		return
	}

	c := cr.Canary()
	if err := c.Validate(); err != nil {
		ch.Context = context.AppendError(ch.Context, v1.ErrorCodeCanaryInvalid.WithDetail(err))
		return
	} else if err = getApp(ch).storage.Save(ch, c); err != nil {
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	context.GetLoggerWithFields(ch, map[interface{}]interface{}{
		"canary.id":  c.ID,
		"canary.ttl": c.TimeToLive,
	}).Print("canary created")

	// TODO: set location header
	w.WriteHeader(http.StatusCreated)
}
