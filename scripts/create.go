package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type canaryRequest struct {
	TimeToLive  int64  `json:"ttl"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func toJson(v interface{}) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		log.Fatal(err)
	}

	return raw
}

func main() {
	hostname := os.Args[1]

	cr := &canaryRequest{
		TimeToLive:  60,
		Title:       os.Args[2],
		Description: "Test Canary",
	}

	data := toJson(cr)
	buf := bytes.NewBuffer(data)
	req, err := http.NewRequest(http.MethodPut, "http://"+hostname+"/canary", buf)
	if err != nil {
		log.Fatal(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("response error: %s", err.Error())
		return
	}

	fmt.Printf("[%d] server >> %s\ndata: \n", res.StatusCode, res.Status)
	rawBody := make([]byte, res.ContentLength)
	_, err = io.ReadFull(res.Body, rawBody)
	fmt.Print(string(rawBody))
	fmt.Println("ok\n")
}
