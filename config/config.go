package config

import (
	"bytes"
	"fmt"
	"github.com/denisschmidt/uploader/constants"
	"github.com/spf13/viper"
)

type Options struct {
	AllowedIPAddresses []string `mapstructure:"allowed_ip_addresses"`
	DefaultUserAgent   string   `mapstructure:"default_user_agent"`
	EnableHealth       bool     `mapstructure:"enable_health"`
	EnableStats        bool     `mapstructure:"enable_stats"`
	EnablePrometheus   bool     `mapstructure:"enable_prometheus"`
}

type Config struct {
	Debug          bool
	Port           int
	DBPath         string
	DBChunkSize    int
	SecretKey      string   `mapstructure:"secret_key"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	Options        *Options
}

func DefaultConfig() *Config {
	return &Config{
		Port:        DefaultPort,
		DBPath:      DefaultDBPath,
		DBChunkSize: DefaultChunkSize,
		Options: &Options{
			DefaultUserAgent: fmt.Sprint(DefaultUserAgent, "/", constants.Version),
		},
	}
}

func load(content string, isPath bool) (*Config, error) {
	config := &Config{}

	defaultConfig := DefaultConfig()

	viper.SetDefault("options", defaultConfig.Options)
	viper.SetDefault("port", defaultConfig.Port)
	viper.SetDefault("dbPath", defaultConfig.DBPath)
	viper.SetDefault("dbChunkSize", defaultConfig.DBChunkSize)
	viper.SetEnvPrefix("uploader")

	var err error

	if isPath == true {
		viper.SetConfigFile(content)
		err = viper.ReadInConfig()
		if err != nil {
			return nil, err
		}
	} else {
		viper.SetConfigType("json")
		err = viper.ReadConfig(bytes.NewBuffer([]byte(content)))
		if err != nil {
			return nil, err
		}
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func Load(path string) (*Config, error) {
	return load(path, true)
}
