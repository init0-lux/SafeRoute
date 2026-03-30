package safety

import (
	"context"
	"math"
	"time"
)

const (
	defaultRadiusMeters       = 500.0
	defaultMaxRadiusMeters    = 3000.0
	defaultRecentWindow       = 6 * time.Hour
	neutralRisk               = 0.25
	recentRiskWeight          = 0.55
	historicalRiskWeight      = 0.20
	timeRiskWeight            = 0.25
	recentWeightSaturation    = 3.0
	historicalWeightSaturaton = 8.0
	confidenceSaturation      = 5.0
)

type Service struct {
	repo Repository
	cfg  ServiceConfig
	now  func() time.Time
}

type ServiceConfig struct {
	DefaultRadiusM float64
	MaxRadiusM     float64
	RecentWindow   time.Duration
}

type ScoreInput struct {
	Latitude  float64
	Longitude float64
	Radius    float64
}

type TimeRisk struct {
	Label      string  `json:"label"`
	Multiplier float64 `json:"multiplier"`
}

type ScoreFactors struct {
	RecentReports         int64   `json:"recent_reports"`
	HistoricalReports     int64   `json:"historical_reports"`
	RecentTrustWeight     float64 `json:"recent_trust_weight"`
	HistoricalTrustWeight float64 `json:"historical_trust_weight"`
	TimeRisk              string  `json:"time_risk"`
	TimeRiskMultiplier    float64 `json:"time_risk_multiplier"`
	Confidence            string  `json:"confidence"`
	ConfidenceScore       float64 `json:"confidence_score"`
	RadiusMeters          float64 `json:"radius_meters"`
	RecentWindowHours     float64 `json:"recent_window_hours"`
}

type ScoreResult struct {
	Score     int          `json:"score"`
	RiskLevel string       `json:"risk_level"`
	Factors   ScoreFactors `json:"factors"`
}

func NewService(repo Repository, cfg ServiceConfig) *Service {
	if cfg.DefaultRadiusM <= 0 {
		cfg.DefaultRadiusM = defaultRadiusMeters
	}
	if cfg.MaxRadiusM <= 0 {
		cfg.MaxRadiusM = defaultMaxRadiusMeters
	}
	if cfg.RecentWindow <= 0 {
		cfg.RecentWindow = defaultRecentWindow
	}

	return &Service{
		repo: repo,
		cfg:  cfg,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) Score(ctx context.Context, input ScoreInput) (*ScoreResult, error) {
	if input.Latitude < -90 || input.Latitude > 90 {
		return nil, ErrInvalidLatitude
	}
	if input.Longitude < -180 || input.Longitude > 180 {
		return nil, ErrInvalidLongitude
	}

	radius := input.Radius
	if radius == 0 {
		radius = s.cfg.DefaultRadiusM
	}
	if radius <= 0 || radius > s.cfg.MaxRadiusM {
		return nil, ErrInvalidRadius
	}

	now := s.now()
	aggregates, err := s.repo.GetAggregates(ctx, AggregateParams{
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		Radius:      radius,
		RecentSince: now.Add(-s.cfg.RecentWindow),
	})
	if err != nil {
		return nil, err
	}

	timeRisk := classifyTimeRisk(now)
	recentComponent := normalize(aggregates.RecentTrustWeight, recentWeightSaturation)
	historicalComponent := normalize(aggregates.HistoricalTrustWeight, historicalWeightSaturaton)
	computedRisk := (recentComponent * recentRiskWeight) +
		(historicalComponent * historicalRiskWeight) +
		(timeRisk.Multiplier * timeRiskWeight)

	confidenceScore := normalize(aggregates.HistoricalTrustWeight+aggregates.RecentTrustWeight, confidenceSaturation)
	risk := blend(neutralRisk, computedRisk, confidenceScore)
	score := int(math.Round(clamp(100-(risk*100), 0, 100)))

	return &ScoreResult{
		Score:     score,
		RiskLevel: riskLevel(score),
		Factors: ScoreFactors{
			RecentReports:         aggregates.RecentReports,
			HistoricalReports:     aggregates.HistoricalReports,
			RecentTrustWeight:     roundFloat(aggregates.RecentTrustWeight),
			HistoricalTrustWeight: roundFloat(aggregates.HistoricalTrustWeight),
			TimeRisk:              timeRisk.Label,
			TimeRiskMultiplier:    roundFloat(timeRisk.Multiplier),
			Confidence:            confidenceLabel(confidenceScore),
			ConfidenceScore:       roundFloat(confidenceScore),
			RadiusMeters:          radius,
			RecentWindowHours:     roundFloat(s.cfg.RecentWindow.Hours()),
		},
	}, nil
}

func classifyTimeRisk(now time.Time) TimeRisk {
	hour := now.Hour()
	switch {
	case hour >= 22 || hour < 5:
		return TimeRisk{Label: "high", Multiplier: 0.80}
	case hour >= 5 && hour < 7:
		return TimeRisk{Label: "moderate", Multiplier: 0.45}
	case hour >= 20 && hour < 22:
		return TimeRisk{Label: "moderate", Multiplier: 0.45}
	default:
		return TimeRisk{Label: "low", Multiplier: 0.10}
	}
}

func normalize(value, saturation float64) float64 {
	if saturation <= 0 {
		return 0
	}
	return clamp(value/saturation, 0, 1)
}

func blend(base, computed, confidence float64) float64 {
	return base + ((computed - base) * clamp(confidence, 0, 1))
}

func riskLevel(score int) string {
	switch {
	case score >= 80:
		return "low"
	case score >= 60:
		return "moderate"
	default:
		return "high"
	}
}

func confidenceLabel(score float64) string {
	switch {
	case score >= 0.75:
		return "high"
	case score >= 0.35:
		return "moderate"
	default:
		return "low"
	}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func roundFloat(value float64) float64 {
	return math.Round(value*100) / 100
}
