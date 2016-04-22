package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/danielkrainas/canaria-api/logging"
)

type StorageConfig struct {
	Driver string `config_default:"true"`
}

type Config struct {
	Storage *StorageConfig
}

func newConfig() *Config {
	config := &Config{}
	config.Storage = &StorageConfig{}
	config.Storage.Driver = "memory"
	return config
}

func validKey(key string, value string) bool {
	if value == "" {
		logging.Trace.Printf("WARNING: key %s should not be empty", key)
		return false
	}

	return true
}

func LoadConfig() (*Config, error) {
	config := newConfig()
	err := config.BindEnv()
	if err == nil {
		err = config.Validate()
	}

	return config, err
}

func (c *Config) BindEnv() error {
	return mapEnvToObject(c, "CANARY")
}

func (c *Config) Validate() error {
	if c.Storage != nil {
		if c.Storage.Driver == "" {
			return errors.New("no storage driver selected")
		}
	}

	return nil
}

func mapEnvToObject(data interface{}, path string) error {
	v := reflect.ValueOf(data)
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fv := v.Field(i)
		ft := fv.Type()

		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		key := fmt.Sprintf("%s_%s", path, strings.ToUpper(f.Name))
		if ft.Kind() != reflect.Struct && strings.ToLower(f.Tag.Get("config_default")) == "true" {
			key = path
		}

		switch ft.Kind() {
		case reflect.Struct:
			err := mapEnvToObject(fv.Interface(), key)
			if err != nil {
				return err
			}

			break

		case reflect.String:
			logging.Trace.Println(key)
			fv.SetString(os.Getenv(key))
			break

		default:
			logging.Warning.Printf("unsupported type for key \"%s\"", key)
			break
		}
	}

	return nil
}
