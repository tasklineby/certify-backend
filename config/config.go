package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Jwt      JwtConfig
}

type ServerConfig struct {
	Host string `mapstructure:"SERVER_HOST"`
	Port int    `mapstructure:"SERVER_PORT"`
}

type DatabaseConfig struct {
	PostgresHost     string `mapstructure:"POSTGRES_HOST"`
	PostgresPort     string `mapstructure:"POSTGRES_PORT"`
	PostgresUser     string `mapstructure:"POSTGRES_USER"`
	PostgresPassword string `mapstructure:"POSTGRES_PASSWORD"`
	PostgresDatabase string `mapstructure:"POSTGRES_DB"`
}

type RedisConfig struct {
	Host     string `mapstructure:"REDIS_HOST"`
	Port     string `mapstructure:"REDIS_PORT"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

type JwtConfig struct {
	AccessTokenSecret string        `mapstructure:"ACCESS_TOKEN_SECRET"`
	AccessTokenTTL    time.Duration `mapstructure:"ACCESS_TOKEN_TTL_MINUTES"`
	RefreshTokenTTL   time.Duration `mapstructure:"REFRESH_TOKEN_TTL_HOURS"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error reading config: %w (searched in: %v)", err, viper.ConfigFileUsed())
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: viper.GetString("SERVER_HOST"),
			Port: viper.GetInt("SERVER_PORT"),
		},
		Database: DatabaseConfig{
			PostgresHost:     viper.GetString("POSTGRES_HOST"),
			PostgresPort:     viper.GetString("POSTGRES_PORT"),
			PostgresUser:     viper.GetString("POSTGRES_USER"),
			PostgresPassword: viper.GetString("POSTGRES_PASSWORD"),
			PostgresDatabase: viper.GetString("POSTGRES_DB"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		Jwt: JwtConfig{
			AccessTokenSecret: viper.GetString("ACCESS_TOKEN_SECRET"),
			AccessTokenTTL:    viper.GetDuration("ACCESS_TOKEN_TTL_MINUTES"),
			RefreshTokenTTL:   viper.GetDuration("REFRESH_TOKEN_TTL_HOURS"),
		},
	}

	return cfg, nil
}

func (c *DatabaseConfig) GetDSN() string {
	result := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.PostgresDatabase)
	return result
}

func (c *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}
