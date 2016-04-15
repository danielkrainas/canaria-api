package main

import (
	"encoding/json"
	"errors"
	"fmt"
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
	ErrMsgDead        = "dead"
)

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
	}

	d.Refresh()
	return d
}

func readcanaryRequest(r *http.Request) (*canaryRequest, error) {
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

func StoreCanary(ctx *context, w http.ResponseWriter, r *http.Request) {
	logging.Trace.Println("StoreCanary: begin")
	defer logging.Trace.Println("StoreCanary: end")
	cr, err := readcanaryRequest(r)
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
	r.HandleFunc("/canary/{id}/{token}", ctx.Contextify(PokeCanary)).Methods("POST")
	r.HandleFunc("/canary/{id}", ctx.Contextify(KillCanary)).Methods("DELETE")
	r.HandleFunc("/canary/{id}", ctx.Contextify(GetCanary)).Methods("HEAD", "GET")
	r.HandleFunc("/canary", ctx.Contextify(StoreCanary)).Methods("PUT")
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	http.ListenAndServe(":6789", loggedRouter)
}
