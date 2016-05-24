package storage

import (
	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/configuration"
	"github.com/danielkrainas/canaria-api/logging"
)

var drivers map[string]StorageDriverFactory = make(map[string]StorageDriverFactory)

func RegisterDriver(key string, factory StorageDriverFactory) {
	drivers[key] = factory
}

type StorageDriverFactory func() StorageDriver

type StorageDriver interface {
	IsDeleted(id string) bool
	Get(id string) (*common.Canary, error)
	Save(c *common.Canary) error
	Delete(id string) error
}

type Storage struct {
	driver StorageDriver
}

func New(storageConfig *configuration.StorageConfig) *Storage {
	factory, ok := drivers[storageConfig.Driver]
	if !ok {
		logging.Error.Fatalf("storage driver \"%s\" not found", storageConfig.Driver)
		return nil
	}

	return &Storage{
		driver: factory(),
	}
}

func (s *Storage) Get(id string) (*common.Canary, error) {
	return s.driver.Get(id)
}

func (s *Storage) Save(c *common.Canary) error {
	return s.driver.Save(c)
}

func (s *Storage) Delete(id string) error {
	return s.driver.Delete(id)
}
