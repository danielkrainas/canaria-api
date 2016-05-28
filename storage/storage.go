package storage

import (
	"fmt"

	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/context"
)

type StorageDriver interface {
	Canaries() CanaryStorage
	Hooks() HookStorage
}

type CanaryStorage interface {
	IsDeleted(ctx context.Context, id string) bool
	Get(ctx context.Context, id string) (*common.Canary, error)
	Store(ctx context.Context, c *common.Canary) error
	Delete(ctx context.Context, id string) error
}

type HookStorage interface {
	Get(ctx context.Context, id string) (*common.WebHook, error)
	Store(ctx context.Context, h *common.WebHook) error
	Delete(ctx context.Context, id string) error
	GetForCanary(ctx context.Context, canaryID string) ([]*common.WebHook, error)
	DeleteForCanary(ctx context.Context, canaryID string) ([]string, error)
}

type Error struct {
	DriverName string
	Enclosed   error
}

func (err Error) Error() string {
	return fmt.Sprintf("%s: %v", err.DriverName, err.Enclosed)
}
