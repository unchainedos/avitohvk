package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type HTTPServer struct {
	ReadTimeout     *time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    *time.Duration `mapstructure:"write_timeout"`
	Addr            string         `mapstructure:"addr"`
	ShutdownTimeout time.Duration  `mapstructure:"shutdown_timeout"`
}

type JWT struct {
	Secret string `mapstructure:"secret"`
}

type Config struct {
	Server HTTPServer `mapstructure:"server"`
	JWT    JWT        `mapstructure:"jwt"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fatal error config file: %w", err)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("marshaling error: %w", err)
	}

	return cfg, nil
}
