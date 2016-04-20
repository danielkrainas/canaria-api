package models

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"

	"github.com/danielkrainas/canaria-api/logging"
	"github.com/satori/go.uuid"
)

const (
	JsonContent = "json"
	FormContent = "form"

	EventDead = "dead"
	EventPing = "ping"

	HookEventHeader     = "X-Canary-Event"
	HookDeliveryHeader  = "X-Canary-Delivery"
	HookSignatureHeader = "X-Canary-Signature"

	HookUserAgent = "Canary-Hooker/0.0.1"
)

type WebHook struct {
	ID          string   `json:"id"`
	ContentType string   `json:"content_type"`
	Secret      string   `json:"secret"`
	InsecureSSL bool     `json:"insecure_ssl"`
	Url         string   `json:"url"`
	Events      []string `json:"events"`
	Active      bool     `json:"active"`
}

type WebHookNotification struct {
	Action string  `json:"action"`
	Canary *Canary `json:"canary"`
}

func NewWebHook() *WebHook {
	return &WebHook{
		ID:     uuid.NewV4().String(),
		Events: []string{},
	}
}

func (wh *WebHook) Validate() error {

	return nil
}

func (wh *WebHook) Deactivate() {
	wh.Active = false
	logging.Warning.Printf("deactivate webhook %s", wh.ID)
}

func (wh *WebHook) Notify(c *Canary, eventType string) error {
	deliveryID := uuid.NewV4()
	n := &WebHookNotification{
		Action: eventType,
		Canary: c,
	}

	req, err := http.NewRequest(http.MethodPost, wh.Url, nil)
	if err != nil {
		logging.Error.Printf("WebHook.Notify: error creating request: %s", err.Error())
		wh.Deactivate()
		return err
	}

	req.Header.Set(HookEventHeader, eventType)
	req.Header.Set(HookDeliveryHeader, deliveryID.String())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", HookUserAgent)

	if wh.ContentType == FormContent {
		form, err := formEncode(n, "")
		if err != nil {
			logging.Error.Printf("WebHook.Notify: error encoding form: %s", err.Error())
			wh.Deactivate()
			return err
		}

		req.Form = form
	} else {
		encoded, err := json.Marshal(n)
		if err != nil {
			logging.Error.Printf("WebHook.Notify: error encoding json: %s", err.Error())
			wh.Deactivate()
			return err
		}

		req.Body = ioutil.NopCloser(bytes.NewBuffer(encoded))
	}

	client := &http.Client{}
	if wh.InsecureSSL {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	res, err := client.Do(req)
	defer res.Body.Close()
	if err != nil || res.StatusCode >= 300 {
		if err == nil {
			err = fmt.Errorf("unexpected status \"%d %s\" encountered", res.StatusCode, res.Status)
		}

		logging.Error.Printf("WebHook.Notify: transmission error: %s", err.Error())
		return err
	}

	return nil
}

func copyFormValues(dest url.Values, src url.Values) {
	for k, v := range src {
		dest.Set(k, v[0])
	}
}

func formEncode(data interface{}, nestPath string) (url.Values, error) {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	form := url.Values{}
	for i := 0; i < t.NumField(); i++ {
		p := t.Field(i)
		if !p.Anonymous {
			name := p.Tag.Get("json")
			if len(nestPath) > 0 {
				name = fmt.Sprintf("%s.%s", nestPath, name)
			}

			switch p.Type.Kind() {
			case reflect.Array:
				af := v.Field(i)
				for j := 0; j < af.Len(); j++ {
					itemValue := af.Index(j)
					if af.Type().Kind() == reflect.Struct {
						return url.Values{}, errors.New("complex objects are not supported at this time.")
					}

					itemForm, err := formEncode(itemValue, name)
					if err != nil {
						return url.Values{}, err
					}

					copyFormValues(form, itemForm)
				}

				break

			case reflect.Struct:
				otherForm, err := formEncode(v.Field(i).Interface(), nestPath)
				if err != nil {
					return url.Values{}, err
				}

				copyFormValues(form, otherForm)
				break

			default:
				form.Set(p.Tag.Get("json"), v.Field(i).String())
				break
			}
		}
	}

	return form, nil
}
