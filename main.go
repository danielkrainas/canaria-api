package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/danielkrainas/canaria-api/configuration"
	"github.com/danielkrainas/canaria-api/context"
	"github.com/danielkrainas/canaria-api/storage"
	_ "github.com/danielkrainas/canaria-api/storage/drivers"

	log "github.com/Sirupsen/logrus"
	ghandlers "github.com/gorilla/handlers"
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

func main() {
	ctx := context.WithVersion(context.Background(), "0.0.1-alpha")

	config, err := configuration.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading configuration: %v\n", err)
		os.Exit(1)
	}

	server, err := newCanaryServer(ctx, config)
	if err != nil {
		log.Fatalln(err)
	}

	if err = server.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}

type CanaryServer struct {
	config *configuration.Config
	app    *handlers.App
	server *http.Server
}

func newCanaryServer(ctx context.Context, config *configuration.Config) (*CanaryServer, error) {
	var err error
	ctx, err = configureLogging(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error configuring logging: %v", err)
	}

	app := handlers.NewApp(ctx, config)
	handler := alive("/", app)
	handler = panicHandler(handler)
	handler = ghandlers.CombinedLoggingHandler(os.Stdout, handler)

	s := &CanaryServer{
		app:    app,
		config: config,
		server: &http.Server{
			Handler: handler,
		},
	}

	return s, nil
}

func configureLogging(ctx context.Context, config *configuration.Config) (context.Context, error) {
	log.SetLevel(log.AllLevels)

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	ctx = context.WithLogger(ctx, context.GetLogger(ctx))
	return ctx, nil
}
