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
	"github.com/danielkrainas/canaria-api/uuid"
)

type canaryHandler struct {
	context.Context
}

func canaryDispatcher(ctx context.Context, r *http.Request) http.Handler {
	ch := &canaryHandler{
		Context: ctx,
	}

	return handlers.MethodHandler{
		"DELETE": http.HandlerFunc(ch.KillCanary),
		"POST":   http.HandlerFunc(ch.UpdateCanary),
		"GET":    http.HandlerFunc(ch.GetCanary),
		"HEAD":   http.HandlerFunc(ch.GetCanary),
	}
}

func canariesDispatcher(ctx context.Context, r *http.Request) http.Handler {
	ch := &canaryHandler{
		Context: ctx,
	}

	return handlers.MethodHandler{
		"PUT": http.HandlerFunc(ch.StoreCanary),
	}
}

type canaryRequest struct {
	TimeToLive   int64    `json:"ttl"`
	Title        string   `json:"title"`
	Message      string   `json:"message"`
	Signature    string   `json:"signature"`
	Tags         []string `json:"tags"`
	PublicKey    string   `json:"pubkey"`
	PublicKeyUrl string   `json:"pubkey_url"`
}

func (r *canaryRequest) Canary() *common.Canary {
	d := &common.Canary{
		ID:         uuid.Generate(),
		TimeToLive: r.TimeToLive,
		Title:      r.Title,
		Message:    r.Message,
		UpdatedAt:  0,
		Tags:       r.Tags,
		Signature:  r.Signature,
	}

	d.Refresh("")
	return d
}

func (ch *canaryHandler) KillCanary(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(ch).Debug("KillCanary")
	c := context.GetCanary(ch)

	context.GetLogger(ch).Warn("killing canary")
	c.Kill()
	if err := getApp(ch).storage.Canaries().Store(ch, c); err != nil {
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	hooks, err := getApp(ch).storage.Hooks().GetForCanary(ch, c.ID)
	if err != nil {
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	for _, wh := range hooks {
		context.GetLogger(ch).Infof("notifying %s of event %s", wh.ID, common.EventDead)
		go actions.Notify(ch, wh, c, common.EventDead)
	}

	if _, err := getApp(ch).storage.Hooks().DeleteForCanary(ch, c.ID); err != nil {
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	for _, wh := range hooks {
		context.GetLogger(ch).Infof("hook removed: %s", wh.ID)
	}

	// TODO: set location header for canary
	w.WriteHeader(http.StatusSeeOther)
}

func (ch *canaryHandler) UpdateCanary(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(ch).Debug("UpdateCanary")
	c := context.GetCanary(ch)

	updateToken := r.Header.Get(common.HeaderCanaryUpdateToken)
	if updateToken == "" || updateToken != c.UpdateToken {
		ch.Context = context.AppendError(ch.Context, v1.ErrorCodeUpdateTokenInvalid.WithDetail(""))
		return
	}

	c.Refresh(updateToken)
	context.GetLogger(ch).Info("update canary")
	if err := getApp(ch).storage.Canaries().Store(ch, c); err != nil {
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	w.Header().Set(common.HeaderCanaryNextUpdateToken, c.UpdateToken)
	w.Header().Set(common.HeaderCanaryID, c.ID)
	common.ServeCanaryJSON(w, c, http.StatusOK)
}

func (ch *canaryHandler) GetCanary(w http.ResponseWriter, r *http.Request) {
	context.GetLogger(ch).Debug("GetCanary")
	c := context.GetCanary(ch)
	w.Header().Set(common.HeaderCanaryID, c.ID)
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusNoContent)
	} else if err := common.ServeCanaryJSON(w, c, http.StatusOK); err != nil {
		context.GetLogger(ch).Errorf("error sending canary json: %v", err)
	}
}

func (ch *canaryHandler) StoreCanary(w http.ResponseWriter, r *http.Request) {
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
	} else if err = getApp(ch).storage.Canaries().Store(ch, c); err != nil {
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	logger := context.GetLoggerWithFields(ch, map[interface{}]interface{}{
		"canary.id":  c.ID,
		"canary.ttl": c.TimeToLive,
	})

	logger.Print("canary created")
	canaryURL, err := getURLBuilder(ch).BuildCanaryURL(c.ID)
	if err != nil {
		logger.Errorf("error building canary url: %v", err)
		ch.Context = context.AppendError(ch.Context, errcode.ErrorCodeUnknown.WithDetail(err))
		return
	}

	w.Header().Set(common.HeaderCanaryID, c.ID)
	w.Header().Set(common.HeaderCanaryNextUpdateToken, c.UpdateToken)
	w.Header().Set("Location", canaryURL)
	w.WriteHeader(http.StatusCreated)
}
