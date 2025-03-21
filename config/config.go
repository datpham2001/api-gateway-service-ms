package config

import "time"

type Config struct {
	Env              string            `yaml:"env" mapstructure:"env"`
	Server           ServerConfig      `yaml:"server" mapstructure:"server"`
	Cache            CacheConfig       `yaml:"cache" mapstructure:"cache"`
	Auth             AuthConfig        `yaml:"auth" mapstructure:"auth"`
	Ratelimit        RatelimitConfig   `yaml:"ratelimit" mapstructure:"ratelimit"`
	FowardServiceUrl map[string]string `yaml:"forward_service_url" mapstructure:"forward_service_url"`
}

type ServerConfig struct {
	Host string `yaml:"host" mapstructure:"host"`
	Port string `yaml:"port" mapstructure:"port"`
	TLS  struct {
		Enable   bool   `yaml:"enable" mapstructure:"enable"`
		CertFile string `yaml:"cert_file" mapstructure:"cert_file"`
		KeyFile  string `yaml:"key_file" mapstructure:"key_file"`
	} `yaml:"tls" mapstructure:"tls"`
}

type CacheConfig struct {
	Host     string `yaml:"host" mapstructure:"host"`
	Port     string `yaml:"port" mapstructure:"port"`
	Password string `yaml:"password" mapstructure:"password"`
	DB       int    `yaml:"db" mapstructure:"db"`
}

type AuthConfig struct {
	JWTSecret     string        `yaml:"jwt_secret" mapstructure:"jwt_secret"`
	JWTExpiration time.Duration `yaml:"jwt_expiration" mapstructure:"jwt_expiration"`
}

type RatelimitConfig struct {
	Limit   int           `yaml:"limit" mapstructure:"limit"`
	Period  time.Duration `yaml:"period" mapstructure:"period"`
	Enabled bool          `yaml:"enabled" mapstructure:"enabled"`
}
