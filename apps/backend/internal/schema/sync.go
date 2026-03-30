package schema

import (
	"saferoute-backend/internal/auth"
	dbconn "saferoute-backend/internal/common/db"
	"saferoute-backend/internal/reports"
	"saferoute-backend/internal/sos"

	"gorm.io/gorm"
)

func Sync(db *gorm.DB) error {
	if err := dbconn.EnsureExtensions(db); err != nil {
		return err
	}

	if err := dbconn.EnsureCustomTypes(db); err != nil {
		return err
	}

	if err := dbconn.EnsureStatusEnums(db); err != nil {
		return err
	}

	if err := db.AutoMigrate(
		&auth.User{},
		&auth.TrustedContact{},
		&auth.UserVerification{},
		&sos.SOSSession{},
		&sos.LocationPing{},
		&reports.Report{},
		&reports.Evidence{},
		&reports.ComplaintEvent{},
	); err != nil {
		return err
	}

	return dbconn.EnsureSchemaArtifacts(db)
}
