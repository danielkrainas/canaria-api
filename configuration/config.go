package configuration

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// TODO: load from TOML
// TODO: load from JSON
// TODO: load from ENV
// TODO: implement marshal/unmarshal for Storage
// TODO: implement marshal/unmarshal for Auth

type Parameters map[string]interface{}

type Storage map[string]Parameters

func (storage Storage) Type() string {
	var storageType []string

	for k := range storage {
		storageType = append(storageType, k)
	}

	if len(storageType) > 1 {
		panic("multiple storage drivers specified in the configuration or environment: " + strings.Join(storageType, ", "))
	}

	if len(storageType) == 1 {
		return storageType[0]
	}

	return ""
}

func (storage Storage) Parameters() Parameters {
	return storage[storage.Type()]
}

func (storage Storage) setParameter(key string, value interface{}) {
	storage[storage.Type()][key] = value
}

type Auth map[string]Parameters

func (auth Auth) Type() string {
	for k := range auth {
		return k
	}

	return ""
}

func (auth Auth) Parameters() Parameters {
	return auth[auth.Type()]
}

func (auth Auth) setParameter(key string, value interface{}) {
	auth[auth.Type()][key] = value
}

type HTTPConfig struct {
	Addr string
	Net  string
	Host string
}

type Config struct {
	Storage Storage

	Auth Auth

	HTTP HTTPConfig
}

func newConfig() *Config {
	config := &Config{
		Auth: make(Auth),

		Storage: make(Storage),

		HTTP: HTTPConfig{
			Addr: ":5678",
			Net:  "tcp",
			Host: "localhost:5678",
		},
	}

	return config
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

		case reflect.String:
			fv.SetString(os.Getenv(key))

		default:
			panic(fmt.Sprintf("unsupported type for key %s: %s", key, ft.Kind().String()))
		}
	}

	return nil
}
