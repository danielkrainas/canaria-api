package common

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type Canary struct {
	ID         string     `json:"id"`
	TimeToLive int64      `json:"ttl"`
	UpdatedAt  int64      `json:"updated_at"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	Tags       []string   `json:"tags"`
	Hooks      []*WebHook `json:"hooks"`
	Signature  string     `json:"signature"`
}

func (c *Canary) Refresh() {
	c.UpdatedAt = time.Now().Unix()
}

func (c *Canary) Kill() {
	c.UpdatedAt = 0
	c.TimeToLive = -1
	c.Title = ""
	c.Message = ""
	c.Tags = []string{}
	c.Signature = ""
}

func (c *Canary) IsZombie() bool {
	t := time.Unix(c.UpdatedAt+c.TimeToLive, 0)
	return time.Now().After(t)
}

func (c *Canary) IsDead() bool {
	return c.TimeToLive < 0
}

func (c *Canary) Validate() error {
	if c.TimeToLive < -1 {
		return errors.New("time to live must be greater than 0")
	}

	return nil
}

func ServeCanaryJSON(w http.ResponseWriter, c *Canary, status int) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(c); err != nil {
		return err
	}

	w.WriteHeader(status)
	return nil
}
