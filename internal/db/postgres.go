package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	// sql driver for database/sql.
	_ "github.com/lib/pq"
	"github.com/mattes/migrate"
	"github.com/mattes/migrate/database"
	"github.com/mattes/migrate/database/postgres"

	// file system migrations.
	_ "github.com/mattes/migrate/source/file"
)

//nolint:gochecknoglobals
var (
	db *sql.DB
)

type Postgres struct {
}

func (p *Postgres) Connect(ctx context.Context, dbURL string, maxConn, maxIdleConn int) (*sql.DB, error) {
	if db != nil {
		return db, nil
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("%w; Unable to open db connection", err)
	}

	if maxConn == 0 {
		maxConn = 25
	}

	db.SetMaxOpenConns(maxConn)
	db.SetMaxIdleConns(maxIdleConn)

	for i := 0; ; i++ {
		err = db.PingContext(ctx)
		if err != nil {
			if i < 10 {
				logrus.WithError(err).Warn("Unable to ping database. Retrying after 1 second")
				time.Sleep(time.Second)

				continue
			}

			return nil, fmt.Errorf("%w; Unable to ping database", err)
		}
		break
	}

	return db, nil
}

func (p *Postgres) Migrate(conn *sql.DB, sourceURL, env string) error {
	driver, err := postgres.WithInstance(conn, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("%w; Unable to create migration instance", err)
	}

	if sourceURL == "" {
		sourceURL = "file://internal/db/migrations/postgres"
	}

	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return fmt.Errorf("%w; Unable to create Migrate instance for database", err)
	}

	version, _, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("%w; unable to get existing migration version for database", err)
	}

	logrus.WithField("version", version).Infoln("current schema version of database")

	if env == "Local" {
		err = m.Drop()
		if err != nil {
			return err
		}

		logrus.Warn("Dropped database")
	}

	for i := 0; i < 5; i++ {
		err = m.Up()
		if err != nil {
			switch err {
			case migrate.ErrNoChange:
				logrus.Infoln("No pending migrations in database")
			case migrate.ErrLocked, database.ErrLocked:
				logrus.Warn("database locked. Assuming another instance working on it. Will retry in a minute")
				time.Sleep(time.Minute)

				continue
			default:
				return fmt.Errorf("%w; error to run migration in database", err)
			}
		}

		break
	}

	newVersion, dirty, err := m.Version()
	if err != nil {
		logrus.WithField("dirty", dirty).
			WithError(err).Panic("unable to get new migration version for database")
	} else if newVersion != version {
		logrus.WithField("old", version).WithField("new", newVersion).Infoln("Migration Successful")
	}

	return nil
}
