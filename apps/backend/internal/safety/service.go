package safety

import (
	"context"
	"math"
	"strings"
	"time"
)

const (
	defaultPointRadiusMeters         = 500.0
	defaultPointMaxRadiusMeters      = 3000.0
	defaultRecentWindow              = 6 * time.Hour
	defaultRouteCorridorRadiusMeters = 75.0
	defaultRouteSegmentLengthMeters  = 150.0
	defaultRouteMaxDistanceMeters    = 10000.0
	minRouteSegments                 = 3
	maxRouteSegments                 = 20

	neutralRisk                = 0.25
	recentRiskWeight           = 0.55
	historicalRiskWeight       = 0.20
	timeRiskWeight             = 0.25
	recentWeightSaturation     = 3.0
	historicalWeightSaturation = 8.0
	confidenceSaturation       = 5.0
	routeAverageRiskWeight     = 0.75
	routeHotspotRiskWeight     = 0.25
	defaultRouteTravelMode     = "walking"
)

type Service struct {
	repo          Repository
	routeProvider RouteProvider
	cfg           ServiceConfig
	now           func() time.Time
	location      *time.Location
}

type ServiceConfig struct {
	DefaultRadiusM       float64
	MaxRadiusM           float64
	RecentWindow         time.Duration
	RouteCorridorRadiusM float64
	RouteSegmentLengthM  float64
	RouteMaxDistanceM    float64
	TimeRisk             TimeRiskConfig
}

type TimeRiskConfig struct {
	Timezone         string
	HighStartHour    int
	HighEndHour      int
	MorningStartHour int
	MorningEndHour   int
	EveningStartHour int
	EveningEndHour   int
}

type Coordinate struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

type ScoreInput struct {
	Latitude  float64
	Longitude float64
	Radius    float64
}

type RouteScoreInput struct {
	Origin      Coordinate
	Destination Coordinate
	TravelMode  string
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

type RouteInfo struct {
	DistanceMeters  int64  `json:"distance_meters"`
	DurationSeconds int64  `json:"duration_seconds"`
	Polyline        string `json:"polyline"`
}

type RouteSummary struct {
	RecentReports           int64   `json:"recent_reports"`
	HistoricalReports       int64   `json:"historical_reports"`
	RecentTrustWeight       float64 `json:"recent_trust_weight"`
	HistoricalTrustWeight   float64 `json:"historical_trust_weight"`
	HighestRiskSegmentIndex int     `json:"highest_risk_segment_index"`
}

type SegmentFactors struct {
	RecentReports         int64   `json:"recent_reports"`
	HistoricalReports     int64   `json:"historical_reports"`
	RecentTrustWeight     float64 `json:"recent_trust_weight"`
	HistoricalTrustWeight float64 `json:"historical_trust_weight"`
	TimeRisk              string  `json:"time_risk"`
	TimeRiskMultiplier    float64 `json:"time_risk_multiplier"`
	Confidence            string  `json:"confidence"`
	ConfidenceScore       float64 `json:"confidence_score"`
}

type RouteSegment struct {
	Index         int            `json:"index"`
	StartFraction float64        `json:"start_fraction"`
	EndFraction   float64        `json:"end_fraction"`
	Score         int            `json:"score"`
	RiskLevel     string         `json:"risk_level"`
	Factors       SegmentFactors `json:"factors"`
}

type RouteScoreResult struct {
	Score     int            `json:"score"`
	RiskLevel string         `json:"risk_level"`
	Route     RouteInfo      `json:"route"`
	Summary   RouteSummary   `json:"summary"`
	Segments  []RouteSegment `json:"segments"`
}

type weightedSignals struct {
	RecentReports         int64
	HistoricalReports     int64
	RecentTrustWeight     float64
	HistoricalTrustWeight float64
}

type riskEvaluation struct {
	Score           int
	Risk            float64
	RiskLevel       string
	TimeRisk        TimeRisk
	Confidence      string
	ConfidenceScore float64
}

func NewService(repo Repository, routeProvider RouteProvider, cfg ServiceConfig) *Service {
	if cfg.DefaultRadiusM <= 0 {
		cfg.DefaultRadiusM = defaultPointRadiusMeters
	}
	if cfg.MaxRadiusM <= 0 {
		cfg.MaxRadiusM = defaultPointMaxRadiusMeters
	}
	if cfg.RecentWindow <= 0 {
		cfg.RecentWindow = defaultRecentWindow
	}
	if cfg.RouteCorridorRadiusM <= 0 {
		cfg.RouteCorridorRadiusM = defaultRouteCorridorRadiusMeters
	}
	if cfg.RouteSegmentLengthM <= 0 {
		cfg.RouteSegmentLengthM = defaultRouteSegmentLengthMeters
	}
	if cfg.RouteMaxDistanceM <= 0 {
		cfg.RouteMaxDistanceM = defaultRouteMaxDistanceMeters
	}
	cfg.TimeRisk = normalizeTimeRiskConfig(cfg.TimeRisk)

	return &Service{
		repo:          repo,
		routeProvider: routeProvider,
		cfg:           cfg,
		now: func() time.Time {
			return time.Now().UTC()
		},
		location: mustLoadLocation(cfg.TimeRisk.Timezone),
	}
}

func (s *Service) Score(ctx context.Context, input ScoreInput) (*ScoreResult, error) {
	return s.ScorePoint(ctx, input)
}

func (s *Service) ScorePoint(ctx context.Context, input ScoreInput) (*ScoreResult, error) {
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
	aggregates, err := s.repo.GetPointAggregates(ctx, PointAggregateParams{
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		Radius:      radius,
		RecentSince: now.Add(-s.cfg.RecentWindow),
	})
	if err != nil {
		return nil, err
	}

	evaluation := s.evaluateRisk(weightedSignals{
		RecentReports:         aggregates.RecentReports,
		HistoricalReports:     aggregates.HistoricalReports,
		RecentTrustWeight:     aggregates.RecentTrustWeight,
		HistoricalTrustWeight: aggregates.HistoricalTrustWeight,
	}, now)

	return &ScoreResult{
		Score:     evaluation.Score,
		RiskLevel: evaluation.RiskLevel,
		Factors: ScoreFactors{
			RecentReports:         aggregates.RecentReports,
			HistoricalReports:     aggregates.HistoricalReports,
			RecentTrustWeight:     roundFloat(aggregates.RecentTrustWeight),
			HistoricalTrustWeight: roundFloat(aggregates.HistoricalTrustWeight),
			TimeRisk:              evaluation.TimeRisk.Label,
			TimeRiskMultiplier:    roundFloat(evaluation.TimeRisk.Multiplier),
			Confidence:            evaluation.Confidence,
			ConfidenceScore:       roundFloat(evaluation.ConfidenceScore),
			RadiusMeters:          radius,
			RecentWindowHours:     roundFloat(s.cfg.RecentWindow.Hours()),
		},
	}, nil
}

func (s *Service) ScoreRoute(ctx context.Context, input RouteScoreInput) (*RouteScoreResult, error) {
	if err := validateOrigin(input.Origin); err != nil {
		return nil, err
	}
	if err := validateDestination(input.Destination); err != nil {
		return nil, err
	}

	travelMode := normalizeTravelMode(input.TravelMode)
	if travelMode != defaultRouteTravelMode {
		return nil, ErrUnsupportedTravelMode
	}
	if s.routeProvider == nil {
		return nil, ErrRouteProviderUnavailable
	}

	route, err := s.routeProvider.ComputeRoute(ctx, RouteRequest{
		Origin:      input.Origin,
		Destination: input.Destination,
		TravelMode:  travelMode,
	})
	if err != nil {
		return nil, err
	}
	if route == nil || len(route.Points) < 2 {
		return nil, ErrRouteNotFound
	}
	if route.DistanceMeters > int64(math.Round(s.cfg.RouteMaxDistanceM)) {
		return nil, ErrRouteTooLong
	}

	lineStringWKT, err := buildLineStringWKT(route.Points)
	if err != nil {
		return nil, err
	}

	now := s.now()
	signals, err := s.repo.GetRouteSignals(ctx, RouteSignalParams{
		LineStringWKT:  lineStringWKT,
		CorridorRadius: s.cfg.RouteCorridorRadiusM,
		RecentSince:    now.Add(-s.cfg.RecentWindow),
	})
	if err != nil {
		return nil, err
	}

	segmentCount := calculateSegmentCount(route.DistanceMeters, s.cfg.RouteSegmentLengthM)
	segmentSignals := make([]weightedSignals, segmentCount)
	summarySignals := weightedSignals{}

	for _, signal := range signals {
		index := segmentIndexForFraction(signal.Fraction, segmentCount)
		segmentSignals[index] = applyRouteSignal(segmentSignals[index], signal)
		summarySignals = applyRouteSignal(summarySignals, signal)
	}

	segments := make([]RouteSegment, 0, segmentCount)
	highestRiskSegmentIndex := 0
	maxRisk := -1.0
	totalRisk := 0.0

	for index := 0; index < segmentCount; index++ {
		startFraction := float64(index) / float64(segmentCount)
		endFraction := float64(index+1) / float64(segmentCount)
		evaluation := s.evaluateRisk(segmentSignals[index], now)
		totalRisk += evaluation.Risk * (endFraction - startFraction)
		if evaluation.Risk > maxRisk {
			maxRisk = evaluation.Risk
			highestRiskSegmentIndex = index
		}

		segments = append(segments, RouteSegment{
			Index:         index,
			StartFraction: roundFraction(startFraction),
			EndFraction:   roundFraction(endFraction),
			Score:         evaluation.Score,
			RiskLevel:     evaluation.RiskLevel,
			Factors: SegmentFactors{
				RecentReports:         segmentSignals[index].RecentReports,
				HistoricalReports:     segmentSignals[index].HistoricalReports,
				RecentTrustWeight:     roundFloat(segmentSignals[index].RecentTrustWeight),
				HistoricalTrustWeight: roundFloat(segmentSignals[index].HistoricalTrustWeight),
				TimeRisk:              evaluation.TimeRisk.Label,
				TimeRiskMultiplier:    roundFloat(evaluation.TimeRisk.Multiplier),
				Confidence:            evaluation.Confidence,
				ConfidenceScore:       roundFloat(evaluation.ConfidenceScore),
			},
		})
	}

	averageRisk := totalRisk
	if maxRisk < 0 {
		maxRisk = neutralRisk
	}
	overallRisk := clamp((averageRisk*routeAverageRiskWeight)+(maxRisk*routeHotspotRiskWeight), 0, 1)
	overallScore := int(math.Round(clamp(100-(overallRisk*100), 0, 100)))

	return &RouteScoreResult{
		Score:     overallScore,
		RiskLevel: riskLevel(overallScore),
		Route: RouteInfo{
			DistanceMeters:  route.DistanceMeters,
			DurationSeconds: route.DurationSeconds,
			Polyline:        route.EncodedPolyline,
		},
		Summary: RouteSummary{
			RecentReports:           summarySignals.RecentReports,
			HistoricalReports:       summarySignals.HistoricalReports,
			RecentTrustWeight:       roundFloat(summarySignals.RecentTrustWeight),
			HistoricalTrustWeight:   roundFloat(summarySignals.HistoricalTrustWeight),
			HighestRiskSegmentIndex: highestRiskSegmentIndex,
		},
		Segments: segments,
	}, nil
}

func validateOrigin(value Coordinate) error {
	if value.Latitude < -90 || value.Latitude > 90 {
		return ErrInvalidOriginLatitude
	}
	if value.Longitude < -180 || value.Longitude > 180 {
		return ErrInvalidOriginLongitude
	}

	return nil
}

func validateDestination(value Coordinate) error {
	if value.Latitude < -90 || value.Latitude > 90 {
		return ErrInvalidDestinationLatitude
	}
	if value.Longitude < -180 || value.Longitude > 180 {
		return ErrInvalidDestinationLongitude
	}

	return nil
}

func normalizeTravelMode(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return defaultRouteTravelMode
	}

	return value
}

func calculateSegmentCount(distanceMeters int64, segmentLengthMeters float64) int {
	if segmentLengthMeters <= 0 {
		segmentLengthMeters = defaultRouteSegmentLengthMeters
	}

	count := int(math.Ceil(float64(distanceMeters) / segmentLengthMeters))
	if count < minRouteSegments {
		return minRouteSegments
	}
	if count > maxRouteSegments {
		return maxRouteSegments
	}

	return count
}

func segmentIndexForFraction(fraction float64, segmentCount int) int {
	if segmentCount <= 1 {
		return 0
	}
	if fraction <= 0 {
		return 0
	}
	if fraction >= 1 {
		return segmentCount - 1
	}

	index := int(math.Floor(fraction * float64(segmentCount)))
	if index >= segmentCount {
		return segmentCount - 1
	}
	if index < 0 {
		return 0
	}

	return index
}

func applyRouteSignal(current weightedSignals, signal RouteSignal) weightedSignals {
	current.HistoricalReports++
	current.HistoricalTrustWeight += signal.TrustWeight
	if signal.IsRecent {
		current.RecentReports++
		current.RecentTrustWeight += signal.TrustWeight
	}

	return current
}

func (s *Service) evaluateRisk(signals weightedSignals, now time.Time) riskEvaluation {
	timeRisk := s.classifyTimeRisk(now)
	recentComponent := normalize(signals.RecentTrustWeight, recentWeightSaturation)
	historicalComponent := normalize(signals.HistoricalTrustWeight, historicalWeightSaturation)
	computedRisk := (recentComponent * recentRiskWeight) +
		(historicalComponent * historicalRiskWeight) +
		(timeRisk.Multiplier * timeRiskWeight)

	confidenceScore := normalize(signals.HistoricalTrustWeight+signals.RecentTrustWeight, confidenceSaturation)
	risk := blend(neutralRisk, computedRisk, confidenceScore)
	score := int(math.Round(clamp(100-(risk*100), 0, 100)))

	return riskEvaluation{
		Score:           score,
		Risk:            risk,
		RiskLevel:       riskLevel(score),
		TimeRisk:        timeRisk,
		Confidence:      confidenceLabel(confidenceScore),
		ConfidenceScore: confidenceScore,
	}
}

func (s *Service) classifyTimeRisk(now time.Time) TimeRisk {
	hour := now.In(s.location).Hour()
	switch {
	case hourInRange(hour, s.cfg.TimeRisk.HighStartHour, s.cfg.TimeRisk.HighEndHour):
		return TimeRisk{Label: "high", Multiplier: 0.80}
	case hourInRange(hour, s.cfg.TimeRisk.MorningStartHour, s.cfg.TimeRisk.MorningEndHour):
		return TimeRisk{Label: "moderate", Multiplier: 0.45}
	case hourInRange(hour, s.cfg.TimeRisk.EveningStartHour, s.cfg.TimeRisk.EveningEndHour):
		return TimeRisk{Label: "moderate", Multiplier: 0.45}
	default:
		return TimeRisk{Label: "low", Multiplier: 0.10}
	}
}

func normalizeTimeRiskConfig(cfg TimeRiskConfig) TimeRiskConfig {
	if strings.TrimSpace(cfg.Timezone) == "" &&
		cfg.HighStartHour == 0 &&
		cfg.HighEndHour == 0 &&
		cfg.MorningStartHour == 0 &&
		cfg.MorningEndHour == 0 &&
		cfg.EveningStartHour == 0 &&
		cfg.EveningEndHour == 0 {
		return TimeRiskConfig{
			Timezone:         "UTC",
			HighStartHour:    22,
			HighEndHour:      5,
			MorningStartHour: 5,
			MorningEndHour:   7,
			EveningStartHour: 20,
			EveningEndHour:   22,
		}
	}

	if strings.TrimSpace(cfg.Timezone) == "" {
		cfg.Timezone = "UTC"
	}

	cfg.HighStartHour = normalizeHour(cfg.HighStartHour, 22)
	cfg.HighEndHour = normalizeHour(cfg.HighEndHour, 5)
	cfg.MorningStartHour = normalizeHour(cfg.MorningStartHour, 5)
	cfg.MorningEndHour = normalizeHour(cfg.MorningEndHour, 7)
	cfg.EveningStartHour = normalizeHour(cfg.EveningStartHour, 20)
	cfg.EveningEndHour = normalizeHour(cfg.EveningEndHour, 22)
	return cfg
}

func normalizeHour(value, fallback int) int {
	if value < 0 || value > 23 {
		return fallback
	}

	return value
}

func hourInRange(hour, start, end int) bool {
	if start == end {
		return true
	}
	if start < end {
		return hour >= start && hour < end
	}

	return hour >= start || hour < end
}

func mustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation(strings.TrimSpace(name))
	if err != nil {
		return time.UTC
	}

	return location
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

func roundFraction(value float64) float64 {
	return math.Round(value*1000) / 1000
}
