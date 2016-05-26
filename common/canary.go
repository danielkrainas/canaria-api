package common

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/satori/go.uuid"
)

type Canary struct {
	ID          string     `json:"id"`
	TimeToLive  int64      `json:"ttl"`
	Refreshed   int64      `json:"refreshed"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	NextToken   string     `json:"next"`
	Tags        []string   `json:"tags"`
	Hooks       []*WebHook `json:"hooks"`
}

func generateNextToken(currentToken string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(uuid.NewV4().String()+currentToken)))
}

func (c *Canary) Refresh() {
	c.Refreshed = time.Now().Unix()
	c.NextToken = generateNextToken(c.NextToken)
}

func (c *Canary) Kill() {
	c.Refreshed = 0
	c.TimeToLive = -1
	c.NextToken = ""
	c.Title = ""
	c.Description = ""
	c.Tags = []string{}
}

func (c *Canary) IsZombie() bool {
	t := time.Unix(c.Refreshed+c.TimeToLive, 0)
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
