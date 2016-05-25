package storage

import (
	"fmt"

	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/context"
)

type StorageDriver interface {
	IsDeleted(ctx context.Context, id string) bool
	Get(ctx context.Context, id string) (*common.Canary, error)
	Save(ctx context.Context, c *common.Canary) error
	Delete(ctx context.Context, id string) error
}

type Error struct {
	DriverName string
	Enclosed   error
}

func (err Error) Error() string {
	return fmt.Sprintf("%s: %v", err.DriverName, err.Enclosed)
}
