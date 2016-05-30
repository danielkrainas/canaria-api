package common

import (
	"github.com/danielkrainas/canaria-api/uuid"
)

const (
	JsonContent = "json"
	FormContent = "form"

	EventDead = "dead"
	EventPing = "ping"
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

func (wh *WebHook) Validate() error {
	return nil
}

func (wh *WebHook) Deactivate() {
	wh.Active = false
}
