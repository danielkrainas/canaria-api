package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	hostname := os.Args[1]

	req, err := http.NewRequest(http.MethodDelete, "http://"+hostname+"/canary/"+os.Args[2], nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("response error: %s", err.Error())
		return
	}

	fmt.Printf("[%d] server >> %s\ndata: \n", res.StatusCode, res.Status)
	if res.ContentLength > 0 {
		rawBody := make([]byte, res.ContentLength)
		_, err = io.ReadFull(res.Body, rawBody)
		fmt.Print(string(rawBody))
	}

	fmt.Println("ok\n")
}
