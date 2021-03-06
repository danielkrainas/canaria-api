package factory

import (
	"fmt"

	"github.com/danielkrainas/canaria-api/storage"
)

var driverFactories = make(map[string]StorageDriverFactory)

type StorageDriverFactory interface {
	Create(parameters map[string]interface{}) (storage.StorageDriver, error)
}

func Register(name string, factory StorageDriverFactory) {
	if factory == nil {
		panic("StorageDriverFactory cannot be nil")
	}

	if _, registered := driverFactories[name]; registered {
		panic(fmt.Sprintf("StorageDriverFactory named %s already registered", name))
	}

	driverFactories[name] = factory
}

func Create(name string, parameters map[string]interface{}) (storage.StorageDriver, error) {
	if factory, ok := driverFactories[name]; ok {
		return factory.Create(parameters)
	}

	return nil, InvalidStorageDriverError{name}
}

type InvalidStorageDriverError struct {
	Name string
}

func (err InvalidStorageDriverError) Error() string {
	return fmt.Sprintf("StorageDriver not registered: %s", err.Name)
}
