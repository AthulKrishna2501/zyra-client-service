package config

import (
	"log"

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
}

func LoadConfig() (cfg Config, err error) {
	viper.SetConfigType("env")

	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err == nil {
		log.Println("Loaded .env from the current directory")
	} else {
		log.Println("Could not load .env from current directory, trying parent...")

		viper.SetConfigFile("../.env")
		if err := viper.ReadInConfig(); err == nil {
			log.Println("Loaded .env from parent directory")
		} else {
			log.Println("Could not load .env from parent directory, trying absolute path...")

			viper.SetConfigFile("/app/.env")
			if err := viper.ReadInConfig(); err == nil {
				log.Println("Loaded .env from absolute path (/app/.env)")
			} else {
				log.Fatalf("Error loading .env file: %v", err)
			}
		}
	}

	viper.AutomaticEnv()

	err = viper.Unmarshal(&cfg)
	return
}
