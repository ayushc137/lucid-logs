package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	App   AppConfig   `koanf:"app"`
	DB    DBConfig    `koanf:"db"`
	HTTP  HTTPConfig  `koanf:"http"`
	Admin AdminConfig `koanf:"admin"`
	JWT   JWTConfig   `koanf:"jwt"`
}

type AppConfig struct {
	Env string `koanf:"env"`
}

type DBConfig struct {
	Host       string `koanf:"host"`
	Port       string `koanf:"port"`
	User       string `koanf:"user"`
	Pass       string `koanf:"pass"`
	Namespace  string `koanf:"namespace"`
	Database   string `koanf:"database"`
	SchemaPath string `koanf:"schema_path"`
}

type HTTPConfig struct {
	Port string `koanf:"port"`
}

type AdminConfig struct {
	Username string `koanf:"username"`
	Password string `koanf:"password"`
}

type JWTConfig struct {
	Secret string `koanf:"secret"`
}

func Load() (*Config, error) {
	k := koanf.New(".")

	// Load from environment variables
	// APP_ENV -> App.Env
	// DB_HOST -> DB.Host
	if err := k.Load(env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	}), nil); err != nil {
		return nil, err
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}

	// Set defaults if needed
	if cfg.HTTP.Port == "" {
		cfg.HTTP.Port = "8080"
	}
	if cfg.DB.Host == "" {
		cfg.DB.Host = "localhost"
	}
	if cfg.DB.Port == "" {
		cfg.DB.Port = "8000"
	}
	if cfg.Admin.Username == "" {
		cfg.Admin.Username = "admin@example.com"
	}
	if cfg.Admin.Password == "" {
		cfg.Admin.Password = "adminadmin"
	}
	if cfg.JWT.Secret == "" {
		cfg.JWT.Secret = generateSecret(32)
		log.Printf("[SECURITY][AUTO-GEN] JWT_SECRET missing; generated temporary secret %q (set JWT_SECRET env to persist)", cfg.JWT.Secret)
	}

	return &cfg, nil
}

func generateSecret(bytesLen int) string {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
