package service

type Config struct {
	HTTPPort int `envconfig:"HTTP_PORT"`
}
