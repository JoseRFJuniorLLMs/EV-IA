package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/app/configs")

	viper.SetEnvPrefix("APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Allow common env vars without APP_ prefix for Docker/VM deploys
	viper.BindEnv("http.port", "HTTP_PORT", "APP_HTTP_PORT")
	viper.BindEnv("database.url", "DATABASE_URL", "APP_DATABASE_URL")
	viper.BindEnv("redis.url", "REDIS_URL", "APP_REDIS_URL")
	viper.BindEnv("nats.url", "NATS_URL", "APP_NATS_URL")
	viper.BindEnv("jwt.secret", "JWT_SECRET", "APP_JWT_SECRET")
	viper.BindEnv("gemini.api_key", "GEMINI_API_KEY", "APP_GEMINI_API_KEY")
	viper.BindEnv("payment.stripe.secret_key", "STRIPE_SECRET_KEY")
	viper.BindEnv("app.environment", "APP_ENVIRONMENT")
	viper.BindEnv("logging.level", "LOG_LEVEL")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// logic for no config file (env vars only) could go here
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
