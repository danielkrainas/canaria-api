package config

import ()

type SillyAuthConfig struct {
	Realm   string
	Service string
}

type HtpasswdAuthConfig struct {
	Realm string
	Path  string
}

type AuthConfig struct {
	Scheme   string `config_default:"true"`
	Silly    *SillyAuthConfig
	Htpasswd *HtpasswdAuthConfig
}
