package storage

import (
	"github.com/danielkrainas/canaria-api/config"
	"github.com/danielkrainas/canaria-api/logging"
	"github.com/danielkrainas/canaria-api/models"
)

var drivers map[string]StorageDriverFactory = make(map[string]StorageDriverFactory)

func RegisterDriver(key string, factory StorageDriverFactory) {
	drivers[key] = factory
}

type StorageDriverFactory func() StorageDriver

type StorageDriver interface {
	Get(id string) (*models.Canary, error)
	Save(c *models.Canary) error
	Delete(id string) error
}

type Storage struct {
	driver StorageDriver
}

func New(storageConfig *config.StorageConfig) *Storage {
	factory, ok := drivers[storageConfig.Driver]
	if !ok {
		logging.Error.Fatalf("storage driver \"%s\" not found", storageConfig.Driver)
		return nil
	}

	return &Storage{
		driver: factory(),
	}
}

func (s *Storage) Get(id string) (*models.Canary, error) {
	return s.driver.Get(id)
}

func (s *Storage) Save(c *models.Canary) error {
	return s.driver.Save(c)
}

func (s *Storage) Delete(id string) error {
	return s.driver.Delete(id)
}
