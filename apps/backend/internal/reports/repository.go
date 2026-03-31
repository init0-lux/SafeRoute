package reports

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, input CreateParams) (*StoredReport, error)
	GetByID(ctx context.Context, id string) (*StoredReport, error)
	ListByUserID(ctx context.Context, userID string) ([]StoredReport, error)
	ListUserHistory(ctx context.Context, userID string) ([]ReportHistoryRow, error)
	ListEvidenceIDs(ctx context.Context, reportID string) ([]string, error)
	ListNearby(ctx context.Context, input NearbyParams) ([]NearbyReportRow, error)
	CountNearby(ctx context.Context, input NearbyParams) (int64, error)
}

type CreateParams struct {
	UserID      *string
	Category    string
	Description *string
	Latitude    float64
	Longitude   float64
	OccurredAt  time.Time
	Source      string
}

type GormRepository struct {
	db *gorm.DB
}

type StoredReport struct {
	ID          string
	UserID      *string
	Category    string
	Description *string
	Latitude    float64
	Longitude   float64
	OccurredAt  time.Time
	CreatedAt   time.Time
	Source      string
	TrustScore  float64
}

type ReportHistoryRow struct {
	StoredReport
	Status string
	Events []ComplaintEventRow
}

type ComplaintEventRow struct {
	ID        string
	Status    string
	Actor     string
	Note      *string
	CreatedAt time.Time
}

type NearbyParams struct {
	Latitude  float64
	Longitude float64
	Radius    float64
	Limit     int
	Offset    int
}

type NearbyReportRow struct {
	StoredReport
	DistanceMeters float64
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, input CreateParams) (*StoredReport, error) {
	var report StoredReport
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := `
			WITH inserted AS (
				INSERT INTO reports (
					user_id,
					category,
					description,
					location,
					occurred_at,
					source
				)
				VALUES (
					?,
					?,
					?,
					ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
					?,
					?
				)
				RETURNING
					id,
					user_id,
					category,
					description,
					location,
					occurred_at,
					created_at,
					source
			)
			SELECT
				i.id,
				i.user_id,
				i.category,
				i.description,
				ST_Y(i.location::geometry) AS latitude,
				ST_X(i.location::geometry) AS longitude,
				i.occurred_at,
				i.created_at,
				i.source,
				COALESCE(u.trust_score, 0.3) AS trust_score
			FROM inserted i
			LEFT JOIN users u ON u.id = i.user_id
		`

		result := tx.Raw(
			query,
			input.UserID,
			input.Category,
			input.Description,
			input.Longitude,
			input.Latitude,
			input.OccurredAt,
			input.Source,
		).Scan(&report)
		if result.Error != nil {
			return result.Error
		}

		// Insert initial "submitted" event
		eventQuery := `
			INSERT INTO complaint_events (report_id, status, actor, note)
			VALUES (?, 'submitted', 'system', 'Incident report received and logged.')
		`
		if err := tx.Exec(eventQuery, report.ID).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if input.UserID != nil && strings.TrimSpace(*input.UserID) != "" {
		report.UserID = input.UserID
	}

	return &report, nil
}

func (r *GormRepository) GetByID(ctx context.Context, id string) (*StoredReport, error) {
	query := `
		SELECT
			r.id,
			r.user_id,
			r.category,
			r.description,
			ST_Y(r.location::geometry) AS latitude,
			ST_X(r.location::geometry) AS longitude,
			r.occurred_at,
			r.created_at,
			r.source,
			COALESCE(u.trust_score, 0.3) AS trust_score
		FROM reports r
		LEFT JOIN users u ON u.id = r.user_id
		WHERE r.id = ?
		LIMIT 1
	`

	var report StoredReport
	result := r.db.WithContext(ctx).Raw(query, id).Scan(&report)
	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, ErrReportNotFound
	}

	return &report, nil
}

func (r *GormRepository) ListByUserID(ctx context.Context, userID string) ([]StoredReport, error) {
	query := `
		SELECT
			r.id,
			r.user_id,
			r.category,
			r.description,
			ST_Y(r.location::geometry) AS latitude,
			ST_X(r.location::geometry) AS longitude,
			r.occurred_at,
			r.created_at,
			r.source,
			COALESCE(u.trust_score, 0.3) AS trust_score
		FROM reports r
		LEFT JOIN users u ON u.id = r.user_id
		WHERE r.user_id = ?
		ORDER BY r.created_at DESC
	`

	var rows []StoredReport
	if err := r.db.WithContext(ctx).Raw(query, userID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *GormRepository) ListUserHistory(ctx context.Context, userID string) ([]ReportHistoryRow, error) {
	// First fetch the reports
	reports, err := r.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(reports) == 0 {
		return []ReportHistoryRow{}, nil
	}

	reportIDs := make([]string, len(reports))
	for i, r := range reports {
		reportIDs[i] = r.ID
	}

	// Fetch all events for these reports
	var eventRows []struct {
		ReportID  string
		ID        string
		Status    string
		Actor     string
		Note      *string
		CreatedAt time.Time
	}

	eventQuery := `
		SELECT id, report_id, status, actor, note, created_at
		FROM complaint_events
		WHERE report_id IN ?
		ORDER BY created_at ASC
	`
	if err := r.db.WithContext(ctx).Raw(eventQuery, reportIDs).Scan(&eventRows).Error; err != nil {
		return nil, err
	}

	// Map events to reports
	eventsByReport := make(map[string][]ComplaintEventRow)
	latestStatusByReport := make(map[string]string)

	for _, er := range eventRows {
		eventsByReport[er.ReportID] = append(eventsByReport[er.ReportID], ComplaintEventRow{
			ID:        er.ID,
			Status:    er.Status,
			Actor:     er.Actor,
			Note:      er.Note,
			CreatedAt: er.CreatedAt,
		})
		latestStatusByReport[er.ReportID] = er.Status
	}

	history := make([]ReportHistoryRow, len(reports))
	for i, rep := range reports {
		history[i] = ReportHistoryRow{
			StoredReport: rep,
			Status:       latestStatusByReport[rep.ID],
			Events:       eventsByReport[rep.ID],
		}
		if history[i].Status == "" {
			history[i].Status = "submitted" // Fallback
		}
	}

	return history, nil
}

func (r *GormRepository) ListEvidenceIDs(ctx context.Context, reportID string) ([]string, error) {
	type evidenceIDRow struct {
		ID string
	}

	var rows []evidenceIDRow
	if err := r.db.WithContext(ctx).
		Raw(`SELECT id FROM evidence WHERE report_id = ? ORDER BY created_at DESC`, reportID).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids, nil
}

func (r *GormRepository) ListNearby(ctx context.Context, input NearbyParams) ([]NearbyReportRow, error) {
	query := `
		SELECT
			r.id,
			r.user_id,
			r.category,
			r.description,
			ST_Y(r.location::geometry) AS latitude,
			ST_X(r.location::geometry) AS longitude,
			r.occurred_at,
			r.created_at,
			r.source,
			ST_Distance(
				r.location,
				ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography
			) AS distance_meters,
			COALESCE(u.trust_score, 0.3) AS trust_score
		FROM reports r
		LEFT JOIN users u ON u.id = r.user_id
		WHERE ST_DWithin(
			r.location,
			ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
			?
		)
		ORDER BY COALESCE(u.trust_score, 0.3) DESC, r.created_at DESC
		LIMIT ? OFFSET ?
	`

	var rows []NearbyReportRow
	if err := r.db.WithContext(ctx).Raw(
		query,
		input.Longitude,
		input.Latitude,
		input.Longitude,
		input.Latitude,
		input.Radius,
		input.Limit,
		input.Offset,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *GormRepository) CountNearby(ctx context.Context, input NearbyParams) (int64, error) {
	type countRow struct {
		Count int64
	}

	query := `
		SELECT COUNT(*) AS count
		FROM reports r
		WHERE ST_DWithin(
			r.location,
			ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
			?
		)
	`

	var row countRow
	if err := r.db.WithContext(ctx).Raw(
		query,
		input.Longitude,
		input.Latitude,
		input.Radius,
	).Scan(&row).Error; err != nil {
		return 0, err
	}

	return row.Count, nil
}
