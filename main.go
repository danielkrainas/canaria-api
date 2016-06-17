package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/danielkrainas/canaria-api/configuration"
	"github.com/danielkrainas/canaria-api/context"
	"github.com/danielkrainas/canaria-api/handlers"
	"github.com/danielkrainas/canaria-api/listener"
	_ "github.com/danielkrainas/canaria-api/storage/memory"

	_ "github.com/danielkrainas/canaria-api/auth/htpasswd"
	_ "github.com/danielkrainas/canaria-api/auth/silly"

	log "github.com/Sirupsen/logrus"
	ghandlers "github.com/gorilla/handlers"
)

func main() {
	ctx := context.WithVersion(context.Background(), "0.0.1-alpha")

	config, err := resolveConfiguration(os.Args[1:])
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

func resolveConfiguration(args []string) (*configuration.Config, error) {
	var configPath string

	if len(args) > 0 {
		configPath = args[0]
	} else if os.Getenv("CANARY_CONFIG_PATH") != "" {
		configPath = os.Getenv("CANARY_CONFIG_PATH")
	}

	if configPath == "" {
		return nil, fmt.Errorf("configuration path not specified")
	}

	fp, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}

	defer fp.Close()
	config, err := configuration.Parse(fp)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", configPath, err)
	}

	return config, nil
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

func (server *CanaryServer) ListenAndServe() error {
	config := server.config
	ln, err := listener.NewListener(config.HTTP.Net, config.HTTP.Addr)
	if err != nil {
		return err
	}

	// TODO: add TLS support
	context.GetLogger(server.app).Infof("listening on %v", ln.Addr())
	return server.server.Serve(ln)
}

func configureLogging(ctx context.Context, config *configuration.Config) (context.Context, error) {

	log.SetLevel(logLevel(config.Log.Level))
	formatter := config.Log.Formatter
	if formatter == "" {
		formatter = "text"
	}

	switch formatter {
	case "json":
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})

	case "text":
		log.SetFormatter(&log.TextFormatter{
			TimestampFormat: time.RFC3339Nano,
		})

	default:
		if config.Log.Formatter != "" {
			return ctx, fmt.Errorf("unsupported log formatter: %q", config.Log.Formatter)
		}
	}

	if config.Log.Formatter != "" {
		log.Debugf("using %q logging formatter", config.Log.Formatter)
	}

	if len(config.Log.Fields) > 0 {
		var fields []interface{}
		for k := range config.Log.Fields {
			fields = append(fields, k)
		}

		ctx = context.WithValues(ctx, config.Log.Fields)
		ctx = context.WithLogger(ctx, context.GetLogger(ctx, fields...))
	}

	ctx = context.WithLogger(ctx, context.GetLogger(ctx))
	return ctx, nil
}

func logLevel(level configuration.LogLevel) log.Level {
	l, err := log.ParseLevel(string(level))
	if err != nil {
		l = log.InfoLevel
		log.Warnf("error parsing level %q: %v, using %q", level, err, l)
	}

	return l
}

func panicHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Panicf("%v", err)
			}
		}()

		handler.ServeHTTP(w, r)
	})
}

func alive(path string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path {
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
