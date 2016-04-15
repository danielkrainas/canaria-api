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

	res, err := http.Post("http://"+hostname+"/canary/"+os.Args[2]+"/"+os.Args[3], "text/plain", nil)
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
