package configs

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/maknahar/alpha-flow/internal/db"

	"github.com/sirupsen/logrus"
)

const (
	defaultTokenValidityDuration = time.Second * 3
)

// Conf contains all the configuration required for the service to run and can be user for dependency ingestion.
type Conf struct {
	// Environment indicates the name of the environment the service will be running on. Default: Production.
	Environment string

	Logger *logrus.Logger

	// Host represents the port on which this service will listen to. Default: :9001
	Host string

	// DB is a database handle representing a connection pool
	DB *sql.DB

	// AccessTokenValidityDuration indicates the time for which a user token is valid for after a successful login. Default: 3s
	AccessTokenValidityDuration time.Duration
}

// Configure reads the env variables for service to start. Default values are set for optional env vars.
func Configure(ctx context.Context) (conf *Conf, err error) {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	conf = &Conf{
		Environment: getEnvOrSetDefault("ENVIRONMENT", "Local"),
		Host:        getEnvOrSetDefault("HOST", ":9001"),
		Logger:      logrus.New(),
	}

	logLevel, err := logrus.ParseLevel(getEnvOrSetDefault("LOG_LEVEL", "Info"))
	if err != nil {
		logrus.Warn("Invalid value set for env var LOG_LEVEL. " +
			"Valid options: Trace, Debug, Info, Warning, Error, Fatal and Panic, Defaulting to Info")

		logLevel = logrus.InfoLevel
	}

	conf.Logger.Level = logLevel
	conf.Logger.SetFormatter(&logrus.JSONFormatter{})

	dbURL := fmt.Sprintf("host=%s sslmode=disable", getEnvOrSetDefault("DB_HOST", "localhost"))

	if getEnvOrSetDefault("DB_USER", "") != "" {
		dbURL += fmt.Sprintf(" user=%s", getEnvOrSetDefault("DB_USER", ""))
	}

	if getEnvOrSetDefault("DB_PASS", "") != "" {
		dbURL += fmt.Sprintf(" password=%s", getEnvOrSetDefault("DB_PASS", ""))
	}

	dbName := getEnvOrSetDefault("DB_NAME", "userapi")

	database := db.New(conf.Environment)

	//if conf.Environment == "local" {
	//	err = database.Recreate(dbURL, dbName)
	//	if err != nil {
	//		return conf, err
	//	}
	//}

	dbMaxConn, err := strconv.Atoi(getEnvOrSetDefault("DB_MAX_CONN", "25"))
	if err != nil {
		logrus.Warn("Invalid value set for env var DB_MAX_CONN. Valid options: Number; " +
			"A rule of thumb is (Max DB connection-10)/max number of instance. Defaulting to 25")

		dbMaxConn = 25
	}

	dbMaxIdleConn, err := strconv.Atoi(getEnvOrSetDefault("DB_MAX_IDLE_CONN", "5"))
	if err != nil {
		logrus.Warn("Invalid value set for env var DB_MAX_IDLE_CONN. " +
			"Valid options: Number less than DB_MAX_CONN. Defaulting to 5")

		dbMaxConn = 5
	}

	dbURL += fmt.Sprintf(" dbname=%s", dbName)

	conf.DB, err = database.Connect(ctx, dbURL, dbMaxConn, dbMaxIdleConn)
	if err != nil {
		return conf, err
	}

	conf.AccessTokenValidityDuration, err = time.ParseDuration(getEnvOrSetDefault("ACCESS_TOKEN_VALIDITY_DURATION", "3s"))
	if err != nil {
		logrus.Warn("Invalid value set for env var TOKEN_VALIDITY_DURATION. " +
			"Valid format: <number><s/m/h>. Defaulting to 3s (3 seconds)")

		conf.AccessTokenValidityDuration = defaultTokenValidityDuration
	}

	return conf, database.Migrate(conf.DB, "", conf.Environment)
}

// getEnvOrSetDefault returns the value set in envVar if exists or defaultValue. Trims any accidental spaces.
func getEnvOrSetDefault(envVar, defaultValue string) string {
	if v, ok := os.LookupEnv(envVar); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}

	logrus.Warnf("No value set for env var %s. Defaulting to %s", envVar, defaultValue)

	return defaultValue
}
