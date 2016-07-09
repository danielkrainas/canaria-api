package common

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/danielkrainas/canaria-api/uuid"
)

const (
	JsonContent = "json"
	FormContent = "form"

	EventDead = "dead"
	EventPing = "ping"
)

var (
	HeaderHookUpdateToken     = "X-Hook-Update-Token"
	HeaderHookNextUpdateToken = "X-Hook-Next-Update-Token"
	HeaderHookID              = "X-Hook-ID"
)

type WebHook struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	ContentType string   `json:"content_type"`
	Secret      string   `json:"secret"`
	InsecureSSL bool     `json:"insecure_ssl"`
	Url         string   `json:"url"`
	Events      []string `json:"events"`
	Active      bool     `json:"active"`
	CanaryID    string   `json:"-"`
	UpdateToken string   `json:"-"`
	UpdatedAt   int64    `json:"updated_at"`
}

type EditHookRequest struct {
	Name   string      `json:"name"`
	Config *HookConfig `json:"config, omitempty"`
	Events []string    `json:"events"`
	Active bool        `json:"active"`
}

func (r *EditHookRequest) Hook() *WebHook {
	hook := NewWebHook()
	hook.Name = r.Name
	hook.Update(r, "")
	return hook
}

type HookConfig struct {
	Url         string `json:"url, omitempty"`
	ContentType string `json:"content_type, omitempty"`
	Secret      string `json:"secret, omitempty"`
	InsecureSSL bool   `json:"insecure_ssl, omitempty"`
}

type WebHookNotification struct {
	Action string  `json:"action"`
	Canary *Canary `json:"canary"`
}

func NewWebHook() *WebHook {
	return &WebHook{
		ID:     uuid.Generate(),
		Events: []string{},
	}
}

func (h *WebHook) Validate() error {
	return nil
}

func (h *WebHook) Deactivate() {
	h.Active = false
}

func (h *WebHook) generateNextToken(lastUpdateToken string) {
	hasher := sha256.New()
	hasher.Write([]byte(lastUpdateToken + strconv.Itoa(int(h.UpdatedAt)) + h.ID))
	h.UpdateToken = base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func (h *WebHook) Update(edit *EditHookRequest, lastUpdateToken string) {
	if edit.Config != nil {
		h.ContentType = edit.Config.ContentType
		h.Secret = edit.Config.Secret
		h.Url = edit.Config.Url
		h.InsecureSSL = edit.Config.InsecureSSL
	}

	h.Active = edit.Active
	if len(edit.Events) > 0 {
		h.Events = make([]string, len(edit.Events))
		for i, e := range edit.Events {
			h.Events[i] = e
		}
	}

	h.UpdatedAt = time.Now().Unix()
	h.generateNextToken(lastUpdateToken)
}

func ServeWebHookJSON(w http.ResponseWriter, h *WebHook, status int) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(h); err != nil {
		return err
	}

	return nil
}
