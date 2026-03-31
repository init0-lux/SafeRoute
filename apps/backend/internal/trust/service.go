package trust

import (
	"context"
	"math"
	"strings"
	"time"

	"saferoute-backend/internal/auth"
)

const (
	baseTrustScore             = 0.30
	corroborationBonusPerMatch = 0.03
	maxCorroborationBonus      = 0.30
	verifiedBonus              = 0.15
)

type Service struct {
	repo Repository
}

type TrustBreakdown struct {
	Base               float64 `json:"base"`
	CorroborationBonus float64 `json:"corroboration_bonus"`
	VerifiedBonus      float64 `json:"verified_bonus"`
}

type Snapshot struct {
	UserID              string         `json:"user_id"`
	Score               float64        `json:"score"`
	ReportsCount        int            `json:"reports_count"`
	CorroborationCount  int            `json:"corroboration_count"`
	Verified            bool           `json:"verified"`
	UpdatedAt           time.Time      `json:"updated_at"`
	Breakdown           TrustBreakdown `json:"breakdown"`
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetByUserID(ctx context.Context, userID string) (*Snapshot, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	user, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	breakdown := calculateBreakdown(*user)
	score := clampTrustScore(
		breakdown.Base +
			breakdown.CorroborationBonus +
			breakdown.VerifiedBonus,
	)
	score = roundTrustScore(score)

	return &Snapshot{
		UserID:             user.ID,
		Score:              score,
		ReportsCount:       user.ReportCount,
		CorroborationCount: user.CorroborationCount,
		Verified:           user.Verified,
		UpdatedAt:          trustUpdatedAt(*user),
		Breakdown:          breakdown,
	}, nil
}

func (s *Service) SetVerification(ctx context.Context, userID string, verified bool) (*Snapshot, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	var verifiedAt *time.Time
	if verified {
		now := time.Now().UTC()
		verifiedAt = &now
	}

	if err := s.repo.SetVerificationStatus(ctx, userID, verified, verifiedAt); err != nil {
		return nil, err
	}

	return s.refreshScore(ctx, userID)
}

func (s *Service) RecordReportSubmission(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrInvalidUserID
	}

	if err := s.repo.IncrementReportCount(ctx, userID); err != nil {
		return err
	}

	_, err := s.refreshScore(ctx, userID)
	return err
}

func (s *Service) refreshScore(ctx context.Context, userID string) (*Snapshot, error) {
	user, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	breakdown := calculateBreakdown(*user)
	score := clampTrustScore(
		breakdown.Base +
			breakdown.CorroborationBonus +
			breakdown.VerifiedBonus,
	)
	score = roundTrustScore(score)

	if err := s.repo.UpdateTrustScore(ctx, userID, score); err != nil {
		return nil, err
	}

	user.TrustScore = score

	return &Snapshot{
		UserID:             user.ID,
		Score:              score,
		ReportsCount:       user.ReportCount,
		CorroborationCount: user.CorroborationCount,
		Verified:           user.Verified,
		UpdatedAt:          trustUpdatedAt(*user),
		Breakdown:          breakdown,
	}, nil
}

func calculateBreakdown(user auth.User) TrustBreakdown {
	corroborationBonus := math.Min(float64(user.CorroborationCount)*corroborationBonusPerMatch, maxCorroborationBonus)
	verifiedBonusValue := 0.0
	if user.Verified {
		verifiedBonusValue = verifiedBonus
	}

	return TrustBreakdown{
		Base:               baseTrustScore,
		CorroborationBonus: corroborationBonus,
		VerifiedBonus:      verifiedBonusValue,
	}
}

func clampTrustScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}

	return score
}

func roundTrustScore(score float64) float64 {
	return math.Round(score*100) / 100
}

func trustUpdatedAt(user auth.User) time.Time {
	if user.VerifiedAt != nil {
		return *user.VerifiedAt
	}

	return user.CreatedAt
}
