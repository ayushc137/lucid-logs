package retrospectives

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/features/analytics"
	"github.com/lucid-logs/go-backend/internal/features/users"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/llm"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the retrospective business logic interface.
type Service interface {
	// List retrieves retrospectives with pagination.
	List(ctx context.Context, userID string, params pagination.Params) (*ListResponse, error)

	// Get retrieves a single retrospective by ID.
	Get(ctx context.Context, id, userID string) (*Retrospective, error)

	// Generate creates a new retrospective with auto-computed summary.
	Generate(ctx context.Context, userID string, req *GenerateRequest) (*Retrospective, error)

	// GenerateDaily generates a daily retrospective (called by scheduler).
	GenerateDaily(ctx context.Context, userID string, date time.Time) (*Retrospective, error)

	// Update updates user content on a retrospective.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Retrospective, error)

	// Delete removes a retrospective.
	Delete(ctx context.Context, id, userID string) error

	// ExistsForToday checks if a retro exists for today.
	ExistsForToday(ctx context.Context, userID string) (bool, error)

	// RegenerateInsights re-runs just the AI insight generation for an existing retro.
	RegenerateInsights(ctx context.Context, id, userID string) (*Retrospective, error)
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo          Repository
	analyticsRepo analytics.Repository
	userRepo      users.Repository
	logger        zerolog.Logger
}

// NewService creates a new retrospectives Service.
func NewService(repo Repository, analyticsRepo analytics.Repository, userRepo users.Repository) Service {
	return &service{
		repo:          repo,
		analyticsRepo: analyticsRepo,
		userRepo:      userRepo,
		logger:        log.With().Str("service", "retrospectives").Logger(),
	}
}

// =============================================================================
// LIST
// =============================================================================

func (s *service) List(ctx context.Context, userID string, params pagination.Params) (*ListResponse, error) {
	retros, total, err := s.repo.FindPaginated(ctx, userID, params.Limit, params.Offset)
	if err != nil {
		s.logger.Error().Err(err).Msg("list retrospectives failed")
		return nil, err
	}

	return &ListResponse{
		Retrospectives: retros,
		Total:          total,
		Limit:          params.Limit,
		Offset:         params.Offset,
	}, nil
}

// =============================================================================
// GET
// =============================================================================

func (s *service) Get(ctx context.Context, id, userID string) (*Retrospective, error) {
	retro, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("retro_id", id).Msg("get retrospective failed")
		return nil, err
	}
	return retro, nil
}

// =============================================================================
// GENERATE
// =============================================================================

func (s *service) Generate(ctx context.Context, userID string, req *GenerateRequest) (*Retrospective, error) {
	start, end := s.resolveDateRange(req)

	// Check if already exists
	existing, err := s.repo.FindByDateRange(ctx, userID, req.RetroType, start, end)
	if err != nil && !errors.Is(err, errors.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		s.logger.Info().Str("retro_id", existing.ID).Msg("retrospective already exists")
		return existing, nil
	}

	// Generate auto-summary
	autoSummary, err := s.computeAutoSummary(ctx, userID, start, end)
	if err != nil {
		s.logger.Error().Err(err).Msg("compute auto summary failed")
		// Continue with empty summary rather than failing
		autoSummary = RetroAutoSummary{}
	}

	// Enhance with AI insights if enabled
	s.enhanceWithAI(ctx, userID, &autoSummary)

	retro := &Retrospective{
		CreatedBy:   userID,
		RetroType:   req.RetroType,
		StartDate:   start,
		EndDate:     end,
		AutoSummary: autoSummary,
		UserContent: UserReflection{},
		Status:      StatusDraft,
		GeneratedAt: time.Now().UTC(),
	}

	created, err := s.repo.Create(ctx, retro)
	if err != nil {
		s.logger.Error().Err(err).Msg("create retrospective failed")
		return nil, err
	}

	s.logger.Info().
		Str("retro_id", created.ID).
		Str("type", req.RetroType).
		Time("start", start).
		Time("end", end).
		Msg("retrospective generated")

	return created, nil
}

// GenerateDaily is called by the scheduler for automatic daily retros.
func (s *service) GenerateDaily(ctx context.Context, userID string, date time.Time) (*Retrospective, error) {
	return s.Generate(ctx, userID, &GenerateRequest{
		RetroType: RetroTypeDaily,
		Date:      &date,
	})
}

// =============================================================================
// UPDATE
// =============================================================================

func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Retrospective, error) {
	retro, err := s.repo.Update(ctx, id, req, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("retro_id", id).Msg("update retrospective failed")
		return nil, err
	}

	s.logger.Info().Str("retro_id", id).Msg("retrospective updated")
	return retro, nil
}

// =============================================================================
// DELETE
// =============================================================================

func (s *service) Delete(ctx context.Context, id, userID string) error {
	err := s.repo.Delete(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("retro_id", id).Msg("delete retrospective failed")
		return err
	}

	s.logger.Info().Str("retro_id", id).Msg("retrospective deleted")
	return nil
}

// =============================================================================
// EXISTS CHECK (for scheduler)
// =============================================================================

func (s *service) ExistsForToday(ctx context.Context, userID string) (bool, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour).Add(-time.Second)

	existing, err := s.repo.FindByDateRange(ctx, userID, RetroTypeDaily, start, end)
	if err != nil {
		return false, err
	}

	return existing != nil, nil
}

// =============================================================================
// AUTO-SUMMARY COMPUTATION
// =============================================================================

func (s *service) computeAutoSummary(ctx context.Context, userID string, start, end time.Time) (RetroAutoSummary, error) {
	summary := RetroAutoSummary{}

	// Get task metrics
	taskMetrics, err := s.analyticsRepo.GetTaskMetrics(ctx, userID, start, end)
	if err != nil {
		s.logger.Warn().Err(err).Msg("get task metrics for retro failed")
	} else {
		summary.Tasks = TasksSummary{
			Completed:          taskMetrics.CompletedTasks,
			Postponed:          taskMetrics.PostponedTasks,
			Canceled:           taskMetrics.AbandonedTasks,
			TotalDurationHours: taskMetrics.TotalDurationHours,
		}
		// Add category breakdown
		for _, cat := range taskMetrics.ByCategory {
			summary.Tasks.ByCategory = append(summary.Tasks.ByCategory, CategoryCount{
				Category:      cat.CategoryName,
				Count:         cat.TaskCount,
				DurationHours: cat.Hours,
			})
		}
	}

	// Get emotion metrics
	emotionMetrics, err := s.analyticsRepo.GetEmotionMetrics(ctx, userID, start, end)
	if err != nil {
		s.logger.Warn().Err(err).Msg("get emotion metrics for retro failed")
	} else {
		summary.Mood = MoodSummary{
			AverageValence:   emotionMetrics.AverageValence,
			AverageArousal:   emotionMetrics.AverageArousal,
			DominantQuadrant: emotionMetrics.DominantQuadrant,
			QuadrantDistribution: QuadrantDist{
				Yellow: emotionMetrics.QuadrantDistribution.Yellow,
				Green:  emotionMetrics.QuadrantDistribution.Green,
				Red:    emotionMetrics.QuadrantDistribution.Red,
				Blue:   emotionMetrics.QuadrantDistribution.Blue,
			},
		}
	}

	// Get goal metrics
	goalMetrics, err := s.analyticsRepo.GetGoalMetrics(ctx, userID)
	if err != nil {
		s.logger.Warn().Err(err).Msg("get goal metrics for retro failed")
	} else {
		// Convert to GoalImpact format
		for _, gp := range goalMetrics.GoalProgress {
			summary.Goals.NetImpact = append(summary.Goals.NetImpact, GoalImpact{
				GoalID: gp.GoalID,
				Name:   gp.GoalTitle,
			})
		}
	}

	// Get category metrics
	categoryMetrics, err := s.analyticsRepo.GetCategoryMetrics(ctx, userID, start, end)
	if err != nil {
		s.logger.Warn().Err(err).Msg("get category metrics for retro failed")
	} else {
		for _, cat := range categoryMetrics.Distribution {
			summary.Categories.TimeDistribution = append(summary.Categories.TimeDistribution, CategoryTime{
				Category:   cat.CategoryName,
				Hours:      cat.Hours,
				Percentage: cat.Percentage,
			})
		}
	}

	return summary, nil
}

// =============================================================================
// HELPERS
// =============================================================================

func (s *service) resolveDateRange(req *GenerateRequest) (start, end time.Time) {
	now := time.Now().UTC()

	switch req.RetroType {
	case RetroTypeDaily:
		date := now
		if req.Date != nil {
			date = *req.Date
		}
		start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		end := start.Add(24 * time.Hour).Add(-time.Second)
		return start, end

	case RetroTypeWeekly:
		// Start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
		end := start.Add(7 * 24 * time.Hour).Add(-time.Second)
		return start, end

	case RetroTypeMonthly:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0).Add(-time.Second)
		return start, end

	case RetroTypeQuarterly:
		quarter := (int(now.Month()) - 1) / 3
		startMonth := time.Month(quarter*3 + 1)
		start := time.Date(now.Year(), startMonth, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 3, 0).Add(-time.Second)
		return start, end

	case RetroTypeYearly:
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)
		return start, end

	case RetroTypeCustom:
		if req.StartDate != nil && req.EndDate != nil {
			return *req.StartDate, *req.EndDate
		}
		fallthrough

	default:
		// Default to today
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		end := start.Add(24 * time.Hour).Add(-time.Second)
		return start, end
	}
}

// =============================================================================
// AI INSIGHT GENERATION
// =============================================================================

// enhanceWithAI enriches the auto summary with AI-generated insights and narrative
// if the user has AI enabled with a valid API key. No-op otherwise (no error).
func (s *service) enhanceWithAI(ctx context.Context, userID string, summary *RetroAutoSummary) {
	aiSettings, err := s.getAIConfig(ctx, userID)
	if err != nil || aiSettings == nil {
		return // AI not configured — silently skip
	}

	insights, narrative, err := s.generateAIInsights(ctx, aiSettings, summary)
	if err != nil {
		s.logger.Warn().Err(err).Msg("AI insight generation failed, continuing without AI")
		return
	}

	summary.Insights = insights
	summary.AINarrative = narrative
}

// getAIConfig fetches and validates the user's AI settings, returning a ready-to-use LLM client.
func (s *service) getAIConfig(ctx context.Context, userID string) (*llm.Client, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	ai := user.Preferences.AI
	if ai == nil || !ai.Enabled || ai.APIKey == "" {
		return nil, nil
	}

	baseURL := llm.ResolveBaseURL(ai.Provider, ai.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("no base URL configured for provider %s", ai.Provider)
	}

	model := ai.Model
	if model == "" {
		model = llm.DefaultModelFor(ai.Provider)
	}

	return llm.NewClient(baseURL, ai.APIKey, model), nil
}

// generateAIInsights calls the LLM with structured retro data and returns insights + narrative.
func (s *service) generateAIInsights(ctx context.Context, client *llm.Client, summary *RetroAutoSummary) ([]string, string, error) {
	// Build a compact JSON representation of the summary stats
	summaryData := buildPromptData(summary)
	summaryJSON, _ := json.Marshal(summaryData)

	systemPrompt := `You are a thoughtful retrospective assistant for a personal journaling app.
Analyze the user's period summary data and provide actionable, specific insights.
Return your response as a JSON object with this exact structure:
{"insights": ["insight 1", "insight 2", ...], "narrative": "a short 2-3 sentence narrative paragraph"}
Provide 3-6 concrete, specific insights. Each insight should reference specific data points.
The narrative should be a warm, honest reflection on the period.
Return ONLY valid JSON, no markdown fences or extra text.`

	userPrompt := fmt.Sprintf("Here is my period summary data:\n\n%s\n\nAnalyze this and provide insights and a narrative.", string(summaryJSON))

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	result, err := client.Complete(ctx, messages)
	if err != nil {
		return nil, "", err
	}

	// Parse JSON response
	type aiResponse struct {
		Insights  []string `json:"insights"`
		Narrative string   `json:"narrative"`
	}

	// Strip markdown fences if present
	cleaned := strings.TrimSpace(result)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var resp aiResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		// If JSON parse fails, use the raw text as narrative and skip insights
		s.logger.Warn().Err(err).Msg("failed to parse AI response as JSON, using raw text")
		return nil, cleaned, nil
	}

	return resp.Insights, resp.Narrative, nil
}

// buildPromptData creates a compact data structure for the LLM prompt.
func buildPromptData(summary *RetroAutoSummary) map[string]any {
	data := map[string]any{}

	// Mood stats
	data["mood"] = map[string]any{
		"average_valence":    summary.Mood.AverageValence,
		"average_arousal":    summary.Mood.AverageArousal,
		"dominant_quadrant":  summary.Mood.DominantQuadrant,
		"quadrant_dist":      summary.Mood.QuadrantDistribution,
		"notable_spikes":     summary.Mood.NotableSpikes,
		"notable_dips":       summary.Mood.NotableDips,
	}

	// Task stats
	data["tasks"] = map[string]any{
		"completed":           summary.Tasks.Completed,
		"postponed":           summary.Tasks.Postponed,
		"canceled":            summary.Tasks.Canceled,
		"total_duration_hours": summary.Tasks.TotalDurationHours,
		"by_category":         summary.Tasks.ByCategory,
	}

	// Categories
	cats := []map[string]any{}
	for _, c := range summary.Categories.TimeDistribution {
		cats = append(cats, map[string]any{
			"category":  c.Category,
			"hours":     c.Hours,
			"percentage": c.Percentage,
		})
	}
	data["categories"] = cats

	// Habits
	data["habits"] = map[string]any{
		"met":           summary.Habits.Met,
		"partially_met": summary.Habits.PartiallyMet,
		"missed":        summary.Habits.Missed,
		"streaks":       summary.Habits.Streaks,
	}

	// Goals
	data["goals"] = map[string]any{
		"net_impact":             summary.Goals.NetImpact,
		"significantly_advanced": summary.Goals.SignificantlyAdvanced,
		"negatively_impacted":    summary.Goals.NegativelyImpacted,
	}

	return data
}

// RegenerateInsights re-runs AI insight generation for an existing retrospective.
func (s *service) RegenerateInsights(ctx context.Context, id, userID string) (*Retrospective, error) {
	retro, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		return nil, err
	}

	client, err := s.getAIConfig(ctx, userID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		// AI not configured — return the retro as-is (no error)
		return retro, nil
	}

	// Clear existing AI content
	retro.AutoSummary.Insights = nil
	retro.AutoSummary.AINarrative = ""

	insights, narrative, err := s.generateAIInsights(ctx, client, &retro.AutoSummary)
	if err != nil {
		s.logger.Warn().Err(err).Msg("regenerate insights failed")
		return nil, errors.ErrInternal.WithMessage("Failed to generate AI insights: " + err.Error())
	}

	retro.AutoSummary.Insights = insights
	retro.AutoSummary.AINarrative = narrative

	return s.repo.UpdateAutoSummary(ctx, id, userID, retro.AutoSummary)
}
