package db

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Needed to run migration
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func Migrate(postgresURL string) error {
	m, err := migrate.New("file://internal/example/db/migrations", postgresURL)
	if err != nil {
		return err
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logrus.WithError(err).Error("migrate failed")
		return err
	}

	return nil
}
