package service

import (
	"github.com/tig-company/featury/internal/audit"
	"github.com/tig-company/featury/internal/models"
)

type diffService struct {
	differ audit.Differ
}

// NewDiffService creates a new diff service
func NewDiffService(differ audit.Differ) DiffService {
	return &diffService{
		differ: differ,
	}
}

// GenerateDiff generates a diff between before and after objects
func (s *diffService) GenerateDiff(before, after interface{}) (models.JSONB, error) {
	return s.differ.GenerateDiff(before, after)
}

// GenerateFeatureFlagDiff generates a specialized diff for feature flags
func (s *diffService) GenerateFeatureFlagDiff(before, after *models.FeatureFlag) (models.JSONB, error) {
	return s.differ.GenerateFeatureFlagDiff(before, after)
}

// GenerateEnvironmentDiff generates a diff for environment configurations
func (s *diffService) GenerateEnvironmentDiff(before, after models.EnvironmentConfig) (models.JSONB, error) {
	return s.differ.GenerateEnvironmentDiff(before, after)
}