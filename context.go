package main

import (
	"net/http"

	"github.com/danielkrainas/canaria-api/storage"
)

type context struct {
	store *storage.Storage
}

type ContextedHandlerFunc func(*context, http.ResponseWriter, *http.Request)

func newContext(store *storage.Storage) *context {
	ctx := &context{
		store: store,
	}

	return ctx
}

func (ctx *context) Contextify(f ContextedHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(ctx, w, r)
	}
}
