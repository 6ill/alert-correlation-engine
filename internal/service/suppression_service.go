package service

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/6ill/alert-correlation-engine/internal/repository"
)

var opsLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		slog.Warn("failed to load Asia/Jakarta timezone, falling back to UTC", "error", err)
		return time.UTC
	}

	return loc
}()

type SuppressionService struct {
	repo *repository.SuppressionRepository
}

func NewSuppressionService(repo *repository.SuppressionRepository) *SuppressionService {
	return &SuppressionService{repo: repo}
}

// Match returns the first non-expired suppression rule that matches this
// alert, or nil if none match. A nil, nil result means "no known pattern and should pass to correlation"
func (s *SuppressionService) Match(ctx context.Context, resourceType, metricType string, at time.Time) (*model.SuppressionRule, error) {
	candidates, err := s.repo.FindCandidates(ctx, resourceType, metricType)
	if err != nil {
		return nil, err
	}

	for _, rule := range candidates {
		if rule.MatchTimeWindow == nil {
			return &rule, nil
		}

		if withinTimeWindow(at.In(opsLocation), *rule.MatchTimeWindow) {
			return &rule, nil
		}
	}

	return nil, nil
}

// withinTimeWindow checks whether `at` falls inside a "HH:MM-HH:MM" window,
// A malformed window fails safe by returning false.
func withinTimeWindow(at time.Time, window string) bool {
	parts := strings.SplitN(window, "-", 2)
	if len(parts) != 2 {
		return false
	}

	startMinutes, okStart := parseClockMinutes(parts[0])
	endMinutes, okEnd := parseClockMinutes(parts[1])
	if !okStart || !okEnd {
		return false
	}

	nowMinutes := at.Hour()*60 + at.Minute()

	if startMinutes <= endMinutes {
		return nowMinutes >= startMinutes && nowMinutes <= endMinutes
	}
	// Window crosses midnight.
	return nowMinutes >= startMinutes || nowMinutes <= endMinutes
}

func parseClockMinutes(hhmm string) (int, bool) {
	hhmm = strings.TrimSpace(hhmm)
	h, m, found := strings.Cut(hhmm, ":")
	if !found {
		return 0, false
	}
	hour, err1 := strconv.Atoi(h)
	minute, err2 := strconv.Atoi(m)
	if err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}

	return hour*60 + minute, true
}
