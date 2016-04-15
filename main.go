package main

import (
	"os"

	"github.com/danielkrainas/canaria-api/config"
	"github.com/danielkrainas/canaria-api/logging"
	"github.com/danielkrainas/canaria-api/storage"
	_ "github.com/danielkrainas/canaria-api/storage/drivers"
)

func main() {
	appConfig, err := config.LoadConfig()
	if err != nil {
		logging.Error.Fatal(err)
		os.Exit(1)
	}

	store := storage.New(appConfig.Storage)
	ctx := newContext(store)
	buildServer(ctx)
}
