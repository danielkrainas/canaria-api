package main

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

	"github.com/danielkrainas/canaria-api/logging"
	"github.com/danielkrainas/canaria-api/models"
)

const (
	ErrMsgBadRequest  = "bad request"
	ErrMsgValidationf = "validation failed: %s"
	ErrMsgDead        = "canary dead"
	ErrMsgBadHookf    = "error executing hook: %s"
)

type WebHookSetupConfig struct {
	Url         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
	InsecureSSL bool   `json:"insecure_ssl"`
}

type WebHookSetupRequest struct {
	Name   string              `json:"name"`
	Config *WebHookSetupConfig `json:"config"`
	Events []string            `json:"events"`
	Active bool                `json:"active"`
}

func (whs *WebHookSetupRequest) Create() *models.WebHook {
	wh := models.NewWebHook()
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

type canaryRequest struct {
	TimeToLive  int64  `json:"ttl"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (r *canaryRequest) ToNewData() *models.Canary {
	d := &models.Canary{
		ID:          uuid.NewV4().String(),
		TimeToLive:  r.TimeToLive,
		Title:       r.Title,
		Description: r.Description,
		Refreshed:   0,
		NextToken:   "",
		Hooks:       []*models.WebHook{},
	}

	d.Refresh()
	return d
}

func readWebHookSetupRequest(r *http.Request) (*WebHookSetupRequest, error) {
	decoder := json.NewDecoder(r.Body)
	whs := &WebHookSetupRequest{}
	err := decoder.Decode(whs)
	return whs, err
}

func readCanaryRequest(r *http.Request) (*canaryRequest, error) {
	decoder := json.NewDecoder(r.Body)
	c := &canaryRequest{}
	err := decoder.Decode(c)
	return c, err
}

func sendCanary(w http.ResponseWriter, c *models.Canary, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	return encoder.Encode(c)
}

func getCanaryTopicId(r *http.Request) (string, error) {
	vars := mux.Vars(r)
	cid := vars["id"]
	if _, err := uuid.FromString(cid); err != nil {
		return "", errors.New("invalid ID")
	}

	return cid, nil
}

func getHookTopicId(r *http.Request) (string, error) {
	vars := mux.Vars(r)
	hid := vars["hookId"]
	if _, err := uuid.FromString(hid); err != nil {
		return "", errors.New("invalid hook ID")
	}

	return hid, nil
}

func loadCanaryTopic(ctx *context, r *http.Request) (*models.Canary, error) {
	cid, err := getCanaryTopicId(r)
	if err != nil {
		return nil, err
	}

	c, err := ctx.store.Get(cid)
	if err != nil {
		// logging
		return nil, nil
	}

	return c, nil
}

func selectHook(c *models.Canary, hookId string) *models.WebHook {
	for _, wh := range c.Hooks {
		if wh.ID == hookId {
			return wh
		}
	}

	return nil
}

func RemoveCanaryHook(ctx *context, w http.ResponseWriter, r *http.Request) {
	logging.Trace.Println("RemoveCanaryHook: begin")
	defer logging.Trace.Println("RemoveCanaryHook: end")
	c, err := loadCanaryTopic(ctx, r)
	if err != nil || c == nil {
		// logging
		http.NotFound(w, r)
		return
	} else if c.IsDead() {
		http.Error(w, ErrMsgDead, http.StatusBadRequest)
		return
	} else if c.IsZombie() {
		c.Kill()
		err := ctx.store.Save(c)
		if err != nil {
			logging.Error.Printf("RemoveCanaryHook: error saving canary: %s", err.Error())
		}

		http.Error(w, ErrMsgDead, http.StatusGone)
		return
	}

	hookId, err := getHookTopicId(r)
	if err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	}

	hookCount := len(c.Hooks)
	if hookCount <= 0 {
		// make error msg var
		http.Error(w, "hook not found", http.StatusNotFound)
		return
	}

	updated := make([]*models.WebHook, hookCount)
	for _, h := range c.Hooks {
		if h.ID != hookId {
			updated = append(updated, h)
		}
	}

	if len(updated) >= hookCount {
		// make error msg var
		http.Error(w, "hook not found", http.StatusNotFound)
		return
	}

	logging.Info.Printf("remove hook %s/%s", c.ID, hookId)
	c.Hooks = updated
	err = ctx.store.Save(c)
	if err != nil {
		logging.Error.Printf("RemoveCanaryHook: error saving canary: %s", err.Error())
	}

	w.WriteHeader(http.StatusNoContent)
}

func CreateCanaryHook(ctx *context, w http.ResponseWriter, r *http.Request) {
	logging.Trace.Println("CreateCanaryHook: begin")
	defer logging.Trace.Println("CreateCanaryHook: end")
	c, err := loadCanaryTopic(ctx, r)
	if err != nil || c == nil {
		// logging
		http.NotFound(w, r)
		return
	} else if c.IsDead() {
		http.Error(w, ErrMsgDead, http.StatusBadRequest)
		return
	} else if c.IsZombie() {
		c.Kill()
		err := ctx.store.Save(c)
		if err != nil {
			logging.Error.Printf("CreateCanaryHook: error saving canary: %s", err.Error())
		}

		http.Error(w, ErrMsgDead, http.StatusGone)
		return
	}

	whs, err := readWebHookSetupRequest(r)
	if err != nil {
		//logging
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	}

	wh := whs.Create()
	c.Hooks = append(c.Hooks, wh)
	if err = wh.Validate(); err != nil {
		http.Error(w, fmt.Sprintf(ErrMsgValidationf, err.Error()), http.StatusBadRequest)
		return
	} else if err = ctx.store.Save(c); err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	}

	logging.Info.Printf("create hook %s/%s", c.ID, wh.ID)
	w.WriteHeader(http.StatusCreated)
	io.WriteString(w, wh.ID)
}

func PingCanaryHook(ctx *context, w http.ResponseWriter, r *http.Request) {
	logging.Trace.Println("PingCanaryHook: begin")
	defer logging.Trace.Println("PingCanaryHook: end")
	c, err := loadCanaryTopic(ctx, r)
	if err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
	} else if c.IsDead() {
		http.Error(w, ErrMsgDead, http.StatusGone)
		return
	} else if c.IsZombie() {
		c.Kill()
		err := ctx.store.Save(c)
		if err != nil {
			logging.Error.Printf("PingCanaryHook: error saving canary: %s", err.Error())
		}

		http.Error(w, ErrMsgDead, http.StatusGone)
		return
	}

	hookId, err := getHookTopicId(r)
	if err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	}

	wh := selectHook(c, hookId)
	if err != nil {
		// logging
		http.NotFound(w, r)
		return
	}

	logging.Info.Printf("ping hook %s/%s", c.ID, hookId)
	err = wh.Notify(c, models.EventPing)
	if err != nil {
		// logging
		http.Error(w, fmt.Sprintf(ErrMsgBadHookf, err.Error()), http.StatusAccepted)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func StoreCanary(ctx *context, w http.ResponseWriter, r *http.Request) {
	logging.Trace.Println("StoreCanary: begin")
	defer logging.Trace.Println("StoreCanary: end")
	cr, err := readCanaryRequest(r)
	if err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	}

	c := cr.ToNewData()
	if err = c.Validate(); err != nil {
		http.Error(w, fmt.Sprintf(ErrMsgValidationf, err.Error()), http.StatusBadRequest)
		return
	} else if err = ctx.store.Save(c); err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	}

	logging.Info.Printf("create %s %d", c.ID, c.TimeToLive)
	if err = sendCanary(w, c, http.StatusCreated); err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
	}
}

func KillCanary(ctx *context, w http.ResponseWriter, r *http.Request) {
	logging.Trace.Println("KillCanary: begin")
	defer logging.Trace.Println("KillCanary: end")
	c, err := loadCanaryTopic(ctx, r)
	if err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	} else if c == nil {
		http.NotFound(w, r)
		return
	} else if c.IsDead() {
		http.Error(w, ErrMsgDead, http.StatusGone)
		return
	}

	logging.Info.Printf("kill %s", c.ID)
	c.Kill()
	err = ctx.store.Save(c)
	if err != nil {
		logging.Error.Printf("KillCanary: error saving canary: %s", err.Error())
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func PokeCanary(ctx *context, w http.ResponseWriter, r *http.Request) {
	logging.Trace.Println("PokeCanary: begin")
	defer logging.Trace.Println("PokeCanary: end")
	c, err := loadCanaryTopic(ctx, r)
	if err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	} else if c == nil || c.IsDead() {
		http.NotFound(w, r)
		return
	} else if c.IsZombie() {
		logging.Info.Printf("kill zombie %s", c.ID)
		c.Kill()
		err := ctx.store.Save(c)
		if err != nil {
			logging.Error.Printf("PokeCanary: error saving canary: %s", err.Error())
		}

		http.NotFound(w, r)
		return
	}

	logging.Info.Printf("refresh %s", c.ID)
	c.Refresh()
	err = ctx.store.Save(c)
	if err != nil {
		logging.Error.Printf("PokeCanary: error saving canary: %s", err.Error())
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	}

	if err = sendCanary(w, c, http.StatusOK); err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
	}
}

func GetCanary(ctx *context, w http.ResponseWriter, r *http.Request) {
	logging.Trace.Println("GetCanary: begin")
	defer logging.Trace.Println("GetCanary: end")
	c, err := loadCanaryTopic(ctx, r)
	if err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
		return
	} else if c == nil {
		http.NotFound(w, r)
		return
	} else if c.IsDead() {
		http.Error(w, ErrMsgDead, http.StatusGone)
		return
	} else if c.IsZombie() {
		c.Kill()
		err := ctx.store.Save(c)
		if err != nil {
			logging.Error.Printf("GetCanary: error saving canary: %s", err.Error())
		}

		http.Error(w, ErrMsgDead, http.StatusGone)
		return
	}

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusNoContent)
	} else if err = sendCanary(w, c, http.StatusOK); err != nil {
		http.Error(w, ErrMsgBadRequest, http.StatusBadRequest)
	}
}

func buildServer(ctx *context) {
	r := mux.NewRouter()
	r.HandleFunc("/canary/{id}/hooks", ctx.Contextify(CreateCanaryHook)).Methods("POST")
	r.HandleFunc("/canary/{id}/hooks/{hookId}", ctx.Contextify(RemoveCanaryHook)).Methods("DELETE")
	r.HandleFunc("/canary/{id}/hooks/{hookId}/ping", ctx.Contextify(PingCanaryHook)).Methods("GET")
	r.HandleFunc("/canary/{id}/{token}", ctx.Contextify(PokeCanary)).Methods("POST")
	r.HandleFunc("/canary/{id}", ctx.Contextify(KillCanary)).Methods("DELETE")
	r.HandleFunc("/canary/{id}", ctx.Contextify(GetCanary)).Methods("HEAD", "GET")
	r.HandleFunc("/canary", ctx.Contextify(StoreCanary)).Methods("PUT")
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	http.ListenAndServe(":6789", loggedRouter)
}
