package drivers

import (
	"errors"

	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/storage"
)

func init() {
	storage.RegisterDriver("memory", memoryDriverFactory)
}

type memoryStorage struct {
	canaries map[string]*common.Canary
}

func memoryDriverFactory() storage.StorageDriver {
	return &memoryStorage{
		canaries: make(map[string]*common.Canary),
	}
}

func (driver *memoryStorage) Get(id string) (*common.Canary, error) {
	c, ok := driver.canaries[id]
	if !ok {
		return nil, errors.New("entry not found")
	}

	return c, nil
}

func (driver *memoryStorage) Save(c *common.Canary) error {
	driver.canaries[c.ID] = c
	return nil
}

func (driver *memoryStorage) Delete(id string) error {
	delete(driver.canaries, id)
	return nil
}

func (driver *memoryStorage) IsDeleted(id string) bool {
	return false
}
