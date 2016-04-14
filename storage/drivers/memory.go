package drivers

import (
	"github.com/danielkrainas/canaria-api/models"
	"github.com/danielkrainas/canaria-api/storage"
)

type memoryStorage struct {
	canaries map[string]*models.Canary
}

func memoryDriverFactory() storage.StorageDriver {
	return &memoryStorage{
		canaries: make(map[string]*models.Canary),
	}
}

func (driver *memoryStorage) Get(id string) (*models.Canary, error) {
	c, ok := canaries[id]
	if !ok {
		return nil, errors.New("entry not found")
	}

	return c, nil
}

func (driver *memoryStorage) Save(c *models.Canary) error {
	canaries[c.ID] = c
	return nil
}

func (driver *memoryStorage) Delete(id string) error {
	delete(driver.canaries, id)
	return nil
}

// may not need
func storageStatEntry(id string) (bool, error) {
	_, ok := canaries[id]
	return ok, nil
}
