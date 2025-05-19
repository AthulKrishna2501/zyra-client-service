package config

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/spf13/viper"
)

type Config struct {
	PORT                  string `mapstructure:"PORT"`
	DB_URL                string `mapstructure:"DB_URL"`
	STRIPE_SECRET_KEY     string `mapstructure:"STRIPE_SECRET_KEY"`
	STRIPE_WEBHOOK_SECRET string `mapstructure:"STRIPE_WEBHOOK_SECRET"`
	ADMIN_EMAIL           string `mapstructure:"ADMIN_EMAIL"`
	STRIPE_SUCCESS_URL    string `mapstructure:"STRIPE_SUCCESS_URL"`
	STRIPE_CANCEL_URL     string `mapstructure:"STRIPE_CANCEL_URL"`
	CLOUD_NAME            string `mapstructure:"CLOUD_NAME"`
	CLOUD_API_KEY         string `mapstructure:"CLOUD_API_KEY"`
	CLOUD_SECRET          string `mapstructure:"CLOUD_SECRET"`
	SECRET_NAME           string `mapstructure:"SECRET_NAME"`
}

func LoadConfig() (cfg Config, err error) {
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	paths := []string{".env", "../.env", "/app/.env"}
	loaded := false

	for _, path := range paths {
		viper.SetConfigFile(path)
		if err := viper.ReadInConfig(); err == nil {
			log.Printf("Loaded configuration from %s", path)
			loaded = true
			break
		} else {
			log.Printf("Failed to load %s: %v", path, err)
		}
	}

	if loaded {
		err = viper.Unmarshal(&cfg)
		if err != nil {
			log.Printf("Failed to unmarshal config from env: %v", err)
			return cfg, err
		}
		log.Printf("Config loaded from env: %+v", cfg)
		return cfg, nil
	}

	log.Println("Falling back to AWS Secrets Manager for configuration")
	secretName := os.Getenv("SECRET_NAME")
	if secretName == "" {
		secretName = "zyra/prod/client-service/env"
	}
	log.Printf("Using secret name: %s", secretName)

	err = loadFromSecretsManager(&cfg, secretName)
	if err != nil {
		log.Printf("Failed to load config from Secrets Manager: %v", err)
		return cfg, err
	}
	log.Printf("Config loaded from Secrets Manager: %+v", cfg)
	return cfg, nil
}

func loadFromSecretsManager(cfg *Config, secretName string) error {
	ctx := context.TODO()

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("Failed to load AWS config: %v", err)
		return err
	}

	client := secretsmanager.NewFromConfig(awsCfg)

	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		log.Printf("Failed to get secret value: %v", err)
		return err
	}

	secretString := *result.SecretString
	log.Printf("Retrieved secret: %s", secretString)
	return json.Unmarshal([]byte(secretString), cfg)
}
