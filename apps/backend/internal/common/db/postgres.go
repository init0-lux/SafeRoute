package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
}

func EnsureExtensions(db *gorm.DB) error {
	statements := []string{
		"CREATE EXTENSION IF NOT EXISTS pgcrypto",
		"CREATE EXTENSION IF NOT EXISTS postgis",
	}

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}

	return nil
}

func EnsureCustomTypes(db *gorm.DB) error {
	statements := []string{
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'sos_session_status') THEN
				CREATE TYPE sos_session_status AS ENUM ('active', 'ended');
			END IF;
		END
		$$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'complaint_event_status') THEN
				CREATE TYPE complaint_event_status AS ENUM ('submitted', 'under_review', 'escalated', 'resolved');
			END IF;
		END
		$$`,
	}

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}

	return nil
}

func EnsureStatusEnums(db *gorm.DB) error {
	statements := []string{
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'sos_sessions'
				  AND column_name = 'status'
				  AND udt_name <> 'sos_session_status'
			) THEN
				ALTER TABLE sos_sessions
					ALTER COLUMN status DROP DEFAULT,
					ALTER COLUMN status TYPE sos_session_status USING status::sos_session_status,
					ALTER COLUMN status SET DEFAULT 'active'::sos_session_status;
			END IF;
		END
		$$`,
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'complaint_events'
				  AND column_name = 'status'
				  AND udt_name <> 'complaint_event_status'
			) THEN
				ALTER TABLE complaint_events
					ALTER COLUMN status TYPE complaint_event_status USING status::complaint_event_status;
			END IF;
		END
		$$`,
	}

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}

	return nil
}

func EnsureSchemaArtifacts(db *gorm.DB) error {
	statements := []string{
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'reports'
				  AND column_name = 'location'
				  AND udt_name <> 'geography'
			) THEN
				ALTER TABLE reports
					ALTER COLUMN location TYPE geography(POINT,4326)
					USING ST_SetSRID(location, 4326)::geography;
			END IF;
		END
		$$`,
		"ALTER TABLE evidence DROP COLUMN IF EXISTS client_encrypted",
		"CREATE INDEX IF NOT EXISTS reports_location_idx ON reports USING GIST(location)",
		"CREATE INDEX IF NOT EXISTS reports_created_at_idx ON reports (created_at DESC)",
		"CREATE INDEX IF NOT EXISTS reports_occurred_at_idx ON reports (occurred_at DESC)",
		"CREATE INDEX IF NOT EXISTS reports_user_created_idx ON reports (user_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS location_pings_session_idx ON location_pings (session_id, recorded_at DESC)",
		"CREATE INDEX IF NOT EXISTS complaint_events_report_created_idx ON complaint_events (report_id, created_at DESC)",
	}

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}

	return nil
}
