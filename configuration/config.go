package configuration

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
)

func (version *Version) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var versionString string
	err := unmarshal(&versionString)
	if err != nil {
		return err
	}

	newVersion := Version(versionString)
	if _, err := newVersion.major(); err != nil {
		return err
	}

	if _, err := newVersion.minor(); err != nil {
		return err
	}

	*version = newVersion
	return nil
}

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

func (storage *Storage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var storageMap map[string]Parameters
	err := unmarshal(&storageMap)
	if err == nil && len(storageMap) > 1 {
		types := make([]string, 0, len(storageMap))
		for k := range storageMap {
			types = append(types, k)
		}

		if len(types) > 1 {
			return fmt.Errorf("Must provide exactly one storage type. provided: %v", types)
		}

		*storage = storageMap
		return nil
	}

	var storageType string
	if err = unmarshal(&storageType); err != nil {
		return err
	}

	*storage = Storage{
		storageType: Parameters{},
	}

	return nil
}

func (storage Storage) MarshalYAML() (interface{}, error) {
	if storage.Parameters() == nil {
		return storage.Type(), nil
	}

	return map[string]Parameters(storage), nil
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

func (auth *Auth) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var m map[string]Parameters
	err := unmarshal(&m)
	if err != nil {
		var authType string
		err = unmarshal(&authType)
		if err == nil {
			*auth = Auth{authType: Parameters{}}
		}

		return err
	}

	if len(m) > 1 {
		types := make([]string, 0, len(m))
		for k := range m {
			types = append(types, k)
		}

		return fmt.Errorf("must specify only one auth type. Provided: %v", types)
	}

	*auth = m
	return nil
}

func (auth Auth) MarshalYAML() (interface{}, error) {
	if auth.Parameters() == nil {
		return auth.Type(), nil
	}

	return map[string]Parameters(auth), nil
}

type HTTPConfig struct {
	Addr    string
	Net     string
	Host    string
	Headers http.Header
}

type LogLevel string

func (logLevel *LogLevel) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strLogLevel string
	err := unmarshal(&strLogLevel)
	if err != nil {
		return err
	}

	strLogLevel = strings.ToLower(strLogLevel)
	switch strLogLevel {
	case "error", "warn", "info", "debug":
	default:
		return fmt.Errorf("Invalid log level %s. Must be one of [error, warn, info, debug]", strLogLevel)
	}

	*logLevel = LogLevel(strLogLevel)
	return nil
}

type LogConfig struct {
	Level     LogLevel
	Formatter string
	Fields    map[string]interface{}
}

type Config struct {
	Log LogConfig

	Storage Storage

	Auth Auth

	HTTP HTTPConfig
}

type v0_1Config Config

func newConfig() *Config {
	config := &Config{
		Log: LogConfig{
			Level:     "debug",
			Formatter: "text",
			Fields:    make(map[string]interface{}),
		},

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

func Parse(rd io.Reader) (*Config, error) {
	in, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	p := NewParser("canary", []VersionedParseInfo{
		{
			Version: MajorMinorVersion(0, 1),
			ParseAs: reflect.TypeOf(v0_1Config{}),
			ConversionFunc: func(c interface{}) (interface{}, error) {
				if v0_1, ok := c.(*v0_1Config); ok {
					if v0_1.Storage.Type() == "" {
						return nil, fmt.Errorf("no storage configuration provided")
					}

					return (*Config)(v0_1), nil
				}

				return nil, fmt.Errorf("Expected *v0_1Config, received %#v", c)
			},
		},
	})

	config := new(Config)
	err = p.Parse(in, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
