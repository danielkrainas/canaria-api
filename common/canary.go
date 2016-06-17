package common

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"
)

var (
	HeaderCanaryUpdateToken     = "X-Canary-Update-Token"
	HeaderCanaryNextUpdateToken = "X-Canary-Next-Update-Token"
	HeaderCanaryID              = "X-Canary-ID"
)

type Canary struct {
	ID           string   `json:"id"`
	TimeToLive   int64    `json:"ttl"`
	UpdatedAt    int64    `json:"updated_at"`
	Title        string   `json:"title"`
	Message      string   `json:"message"`
	Tags         []string `json:"tags"`
	Signature    string   `json:"signature"`
	PublicKey    string   `json:"pubkey"`
	PublicKeyUrl string   `json:"pubkey_url"`
	UpdateToken  string   `json:"-"`
}

func (c *Canary) Refresh(lastUpdateToken string) {
	c.UpdatedAt = time.Now().Unix()
	hasher := sha256.New()
	hasher.Write([]byte(lastUpdateToken + strconv.Itoa(int(c.UpdatedAt)) + c.ID))
	c.UpdateToken = base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func (c *Canary) Kill() {
	c.UpdatedAt = 0
	c.TimeToLive = -1
	c.Title = ""
	c.Message = ""
	c.Tags = []string{}
	c.Signature = ""
	c.UpdateToken = ""
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
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(c); err != nil {
		return err
	}

	return nil
}
