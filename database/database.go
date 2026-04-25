package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(driver, uri string) (*gorm.DB, error) {
	switch driver {
	case "sqlite":
		return openSQLite(uri)
	case "postgres":
		return openPostgres(uri)
	default:
		return nil, nil
	}
}

func Migrate(db *gorm.DB, models ...interface{}) error {
	return db.AutoMigrate(models...)
}

func openPostgres(uri string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(uri), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func openSQLite(uri string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(uri), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
