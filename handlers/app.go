package handlers

import (
	"github.com/danielkrainas/canaria-api/auth"
	"github.com/danielkrainas/canaria-api/config"
	"github.com/danielkrainas/canaria-api/storage"

	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

type App struct {
	context.Context

	Config       *config.Config
	router       *mux.Router
	storage      storage.StorageDriver
	authStrategy auth.AuthStrategy

	readOnly bool
}
