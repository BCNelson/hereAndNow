package filters

import (
	"fmt"

	"github.com/bcnelson/hereAndNow/pkg/models"
)

type PriorityFilter struct {
	config FilterConfig
}

type PriorityScore struct {
	Task           models.Task `json:"task"`
	TotalScore     float64     `json:"total_score"`
	PriorityScore  float64     `json:"priority_score"`
	UrgencyScore   float64     `json:"urgency_score"`
	ContextScore   float64     `json:"context_score"`
	EnergyScore    float64     `json:"energy_score"`
	Explanation    string      `json:"explanation"`
}

func NewPriorityFilter(config FilterConfig) *PriorityFilter {
	return &PriorityFilter{
		config: config,
	}
}

func (f *PriorityFilter) Name() string {
	return "priority"
}

func (f *PriorityFilter) Priority() int {
	return 80
}

func (f *PriorityFilter) Apply(ctx models.Context, task models.Task) (visible bool, reason string) {
	if !f.config.EnablePriorityFilter {
		return true, "priority filtering disabled"
	}

	score := f.CalculatePriorityScore(ctx, task)

	threshold := f.calculateDynamicThreshold(ctx)
	
	if score.TotalScore >= threshold {
		return true, fmt.Sprintf("priority score %.1f >= threshold %.1f (%s)", 
			score.TotalScore, threshold, score.Explanation)
	}

	return false, fmt.Sprintf("priority score %.1f < threshold %.1f (%s)", 
		score.TotalScore, threshold, score.Explanation)
}

func (f *PriorityFilter) CalculatePriorityScore(ctx models.Context, task models.Task) PriorityScore {
	priorityScore := f.calculateTaskPriorityScore(task)
	urgencyScore := f.calculateUrgencyScore(ctx, task)
	contextScore := f.calculateContextScore(ctx, task)
	energyScore := f.calculateEnergyMatchScore(ctx, task)

	weights := f.getScoreWeights(ctx)
	totalScore := (priorityScore * weights.Priority) +
		(urgencyScore * weights.Urgency) +
		(contextScore * weights.Context) +
		(energyScore * weights.Energy)

	explanation := fmt.Sprintf("P:%.1f×%.1f + U:%.1f×%.1f + C:%.1f×%.1f + E:%.1f×%.1f",
		priorityScore, weights.Priority,
		urgencyScore, weights.Urgency,
		contextScore, weights.Context,
		energyScore, weights.Energy)

	return PriorityScore{
		Task:           task,
		TotalScore:     totalScore,
		PriorityScore:  priorityScore,
		UrgencyScore:   urgencyScore,
		ContextScore:   contextScore,
		EnergyScore:    energyScore,
		Explanation:    explanation,
	}
}

type ScoreWeights struct {
	Priority float64
	Urgency  float64
	Context  float64
	Energy   float64
}

func (f *PriorityFilter) getScoreWeights(ctx models.Context) ScoreWeights {
	baseWeights := ScoreWeights{
		Priority: 0.4,
		Urgency:  0.3,
		Context:  0.2,
		Energy:   0.1,
	}

	if ctx.AvailableMinutes < 30 {
		baseWeights.Urgency += 0.1
		baseWeights.Priority -= 0.1
	}

	if ctx.EnergyLevel <= 2 {
		baseWeights.Energy += 0.15
		baseWeights.Context -= 0.15
	}

	return baseWeights
}

func (f *PriorityFilter) calculateTaskPriorityScore(task models.Task) float64 {
	return float64(task.Priority) / 10.0
}

func (f *PriorityFilter) calculateUrgencyScore(ctx models.Context, task models.Task) float64 {
	if task.DueAt == nil {
		return 0.5
	}

	timeUntilDue := task.DueAt.Sub(ctx.Timestamp)
	hoursUntilDue := timeUntilDue.Hours()

	switch {
	case hoursUntilDue <= 0:
		return 1.0
	case hoursUntilDue <= 2:
		return 0.9
	case hoursUntilDue <= 6:
		return 0.8
	case hoursUntilDue <= 24:
		return 0.6
	case hoursUntilDue <= 72:
		return 0.4
	case hoursUntilDue <= 168:
		return 0.2
	default:
		return 0.1
	}
}

func (f *PriorityFilter) calculateContextScore(ctx models.Context, task models.Task) float64 {
	score := 0.5

	if task.EstimatedMinutes != nil && ctx.AvailableMinutes > 0 {
		timeMatch := float64(ctx.AvailableMinutes) / float64(*task.EstimatedMinutes)
		if timeMatch >= 1.0 {
			score += 0.3
		} else if timeMatch >= 0.5 {
			score += 0.1
		}
	}

	socialBonus := f.calculateSocialContextBonus(ctx, task)
	score += socialBonus

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (f *PriorityFilter) calculateSocialContextBonus(ctx models.Context, task models.Task) float64 {
	switch ctx.SocialContext {
	case models.SocialContextAtWork:
		if f.isWorkRelatedTask(task) {
			return 0.2
		}
		return -0.1
	case models.SocialContextWithFamily:
		if f.isFamilyRelatedTask(task) {
			return 0.2
		}
		if f.isWorkRelatedTask(task) {
			return -0.2
		}
		return 0.0
	case models.SocialContextAlone:
		if f.isFocusTask(task) {
			return 0.15
		}
		return 0.0
	default:
		return 0.0
	}
}

func (f *PriorityFilter) calculateEnergyMatchScore(ctx models.Context, task models.Task) float64 {
	requiredEnergy := f.estimateRequiredEnergy(task)
	if requiredEnergy <= ctx.EnergyLevel {
		return 1.0
	}

	energyDeficit := requiredEnergy - ctx.EnergyLevel
	penalty := float64(energyDeficit) * 0.2
	score := 1.0 - penalty

	if score < 0.0 {
		score = 0.0
	}

	return score
}

func (f *PriorityFilter) estimateRequiredEnergy(task models.Task) int {
	baseEnergy := 1

	if task.EstimatedMinutes != nil {
		minutes := *task.EstimatedMinutes
		switch {
		case minutes > 120:
			baseEnergy = 4
		case minutes > 60:
			baseEnergy = 3
		case minutes > 30:
			baseEnergy = 2
		}
	}

	if task.Priority >= 8 {
		baseEnergy++
	}

	if f.isComplexTask(task) {
		baseEnergy++
	}

	if baseEnergy > 5 {
		baseEnergy = 5
	}

	return baseEnergy
}

func (f *PriorityFilter) calculateDynamicThreshold(ctx models.Context) float64 {
	baseThreshold := 0.5

	if ctx.AvailableMinutes < 15 {
		baseThreshold += 0.2
	} else if ctx.AvailableMinutes > 120 {
		baseThreshold -= 0.1
	}

	if ctx.EnergyLevel <= 2 {
		baseThreshold += 0.15
	} else if ctx.EnergyLevel >= 4 {
		baseThreshold -= 0.1
	}

	if ctx.SocialContext == models.SocialContextAtWork {
		baseThreshold -= 0.05
	}

	hour := ctx.Timestamp.Hour()
	if hour >= 6 && hour <= 10 {
		baseThreshold -= 0.1
	} else if hour >= 22 || hour <= 5 {
		baseThreshold += 0.2
	}

	if baseThreshold < 0.1 {
		baseThreshold = 0.1
	} else if baseThreshold > 0.9 {
		baseThreshold = 0.9
	}

	return baseThreshold
}

func (f *PriorityFilter) isWorkRelatedTask(task models.Task) bool {
	workKeywords := []string{"meeting", "email", "report", "project", "work", "client", "deadline"}
	taskText := fmt.Sprintf("%s %s", task.Title, task.Description)
	
	for _, keyword := range workKeywords {
		if containsIgnoreCase(taskText, keyword) {
			return true
		}
	}
	return false
}

func (f *PriorityFilter) isFamilyRelatedTask(task models.Task) bool {
	familyKeywords := []string{"family", "kids", "home", "personal", "grocery", "appointment", "pickup", "school"}
	taskText := fmt.Sprintf("%s %s", task.Title, task.Description)
	
	for _, keyword := range familyKeywords {
		if containsIgnoreCase(taskText, keyword) {
			return true
		}
	}
	return false
}

func (f *PriorityFilter) isFocusTask(task models.Task) bool {
	focusKeywords := []string{"study", "read", "write", "plan", "research", "design", "code", "analyze"}
	taskText := fmt.Sprintf("%s %s", task.Title, task.Description)
	
	for _, keyword := range focusKeywords {
		if containsIgnoreCase(taskText, keyword) {
			return true
		}
	}
	return false
}

func (f *PriorityFilter) isComplexTask(task models.Task) bool {
	complexKeywords := []string{"complex", "difficult", "challenging", "research", "analysis", "design", "architecture"}
	taskText := fmt.Sprintf("%s %s", task.Title, task.Description)
	
	for _, keyword := range complexKeywords {
		if containsIgnoreCase(taskText, keyword) {
			return true
		}
	}
	
	return task.EstimatedMinutes != nil && *task.EstimatedMinutes > 60
}

func containsIgnoreCase(text, substr string) bool {
	return len(text) >= len(substr) && 
		   len(findIgnoreCase(text, substr)) > 0
}

func findIgnoreCase(text, substr string) string {
	lowerText := toLower(text)
	lowerSubstr := toLower(substr)
	
	for i := 0; i <= len(lowerText)-len(lowerSubstr); i++ {
		if lowerText[i:i+len(lowerSubstr)] == lowerSubstr {
			return text[i : i+len(lowerSubstr)]
		}
	}
	return ""
}

func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}