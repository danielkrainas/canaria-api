package memory

import (
	"errors"

	"github.com/danielkrainas/canaria-api/common"
	"github.com/danielkrainas/canaria-api/context"
	"github.com/danielkrainas/canaria-api/storage"
	"github.com/danielkrainas/canaria-api/storage/factory"
)

const driverName = "memory"

func init() {
	factory.Register(driverName, &memoryDriverFactory{})
}

type memoryDriverFactory struct{}

func (factory *memoryDriverFactory) Create(parameters map[string]interface{}) (storage.StorageDriver, error) {
	return New(), nil
}

type driver struct {
	canaries map[string]*common.Canary
	deleted  map[string]*common.Canary
}

func New() *driver {
	return &driver{
		deleted:  make(map[string]*common.Canary),
		canaries: make(map[string]*common.Canary),
	}
}

func (d *driver) Get(ctx context.Context, id string) (*common.Canary, error) {
	c, ok := d.canaries[id]
	if !ok {
		return nil, errors.New("entry not found")
	}

	return c, nil
}

func (d *driver) Save(ctx context.Context, c *common.Canary) error {
	d.canaries[c.ID] = c
	return nil
}

func (d *driver) Delete(ctx context.Context, id string) error {
	if c, ok := d.canaries[id]; ok {
		delete(d.canaries, id)
		d.deleted[id] = c
	}

	return nil
}

func (d *driver) IsDeleted(ctx context.Context, id string) bool {
	_, found := d.deleted[id]
	return found
}
