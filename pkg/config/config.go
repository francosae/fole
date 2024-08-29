package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
)

type Config struct {
	Port                    string `mapstructure:"PORT"`
	Username                string `mapstructure:"USERNAME"`
	Password                string `mapstructure:"PASSWORD"`
	Host                    string `mapstructure:"HOST"`
	DbPort                  string `mapstructure:"DBPORT"`
	FirebaseCredentialsPath string `mapstructure:"FIREBASE_PATH"`
	BackupFirebasePath      string `mapstructure:"BACKUP_FIREBASE_PATH"`
	RedisCredentialsPath    string `mapstructure:"REDIS_PATH"`
	ConnectionString        string
	AlgoliaKey              string `mapstructure:"ALGOLIA_KEY"`
	AlgoliaAppId            string `mapstructure:"ALGOLIA_APP_ID"`
	SupabaseProjectURL      string `mapstructure:"SUPABASE_PROJECT_URL"`
	SupabaseAPIKey          string `mapstructure:"SUPABASE_API_KEY"`
}

func getConfigValue(key string) string {
	value := viper.GetString(key)
	if value == "" {
		value = os.Getenv(key)
	}
	return value
}

func LoadConfig() (config Config, err error) {

	viper.AddConfigPath("./pkg/config/envs")
	viper.AddConfigPath("../pkg/config/envs")
	viper.AddConfigPath("/app/config")
	viper.AddConfigPath("/app/pkg/config/envs")
	viper.SetConfigName("prod")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()

	if err != nil {
		return
	}

	host := getConfigValue("HOST")
	port := getConfigValue("DBPORT")
	user := getConfigValue("USERNAME")
	password := getConfigValue("PASSWORD")

	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s",
		host, port, user, password,
	)
	config.ConnectionString = connectionString

	err = viper.Unmarshal(&config)

	return
}
