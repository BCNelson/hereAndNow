package filters

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
)

type Engine struct {
	rules       []FilterRule
	auditRepo   FilterAuditRepository
	config      FilterConfig
	mu          sync.RWMutex
}

type FilterAuditRepository interface {
	SaveFilterResult(audit models.FilterAudit) error
	GetAuditLogByTaskID(taskID string, limit int) ([]models.FilterAudit, error)
	GetAuditLogByUserID(userID string, since time.Time, limit int) ([]models.FilterAudit, error)
}

func NewEngine(config FilterConfig, auditRepo FilterAuditRepository) *Engine {
	return &Engine{
		rules:     []FilterRule{},
		auditRepo: auditRepo,
		config:    config,
	}
}

func (e *Engine) AddRule(rule FilterRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	for i, existingRule := range e.rules {
		if existingRule.Name() == rule.Name() {
			e.rules[i] = rule
			return
		}
	}
	
	e.rules = append(e.rules, rule)
	e.sortRulesByPriority()
}

func (e *Engine) RemoveRule(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	for i, rule := range e.rules {
		if rule.Name() == name {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return
		}
	}
}

func (e *Engine) FilterTasks(ctx models.Context, tasks []models.Task) ([]models.Task, []FilterResult) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	visibleTasks := []models.Task{}
	allResults := []FilterResult{}
	
	for _, task := range tasks {
		visible, results := e.evaluateTask(ctx, task)
		allResults = append(allResults, results...)
		
		if visible {
			visibleTasks = append(visibleTasks, task)
		}
	}
	
	e.auditFilterResults(ctx, allResults)
	
	return visibleTasks, allResults
}

func (e *Engine) evaluateTask(ctx models.Context, task models.Task) (bool, []FilterResult) {
	results := []FilterResult{}
	overallVisible := true
	
	for _, rule := range e.rules {
		visible, reason := rule.Apply(ctx, task)
		
		result := FilterResult{
			TaskID:     task.ID,
			Visible:    visible,
			Reason:     reason,
			FilterName: rule.Name(),
		}
		results = append(results, result)
		
		if !visible {
			overallVisible = false
		}
	}
	
	return overallVisible, results
}

func (e *Engine) GetAuditLog(taskID string, ctx models.Context) ([]FilterResult, error) {
	audits, err := e.auditRepo.GetAuditLogByTaskID(taskID, 50)
	if err != nil {
		return nil, fmt.Errorf("error retrieving audit log: %v", err)
	}
	
	results := []FilterResult{}
	for _, audit := range audits {
		result := FilterResult{
			TaskID:     audit.TaskID,
			Visible:    audit.IsVisible,
			Reason:     string(audit.Reasons),
			FilterName: "audit",
		}
		results = append(results, result)
	}
	
	return results, nil
}

func (e *Engine) auditFilterResults(ctx models.Context, results []FilterResult) {
	for _, result := range results {
		reason := models.FilterReason{
			Rule:    result.FilterName,
			Passed:  result.Visible,
			Details: result.Reason,
		}
		reasonJSON, _ := json.Marshal([]models.FilterReason{reason})
		
		audit := models.FilterAudit{
			ID:            generateAuditID(),
			TaskID:        result.TaskID,
			UserID:        ctx.UserID,
			ContextID:     "",
			IsVisible:     result.Visible,
			Reasons:       reasonJSON,
			PriorityScore: 0.0,
			CreatedAt:     ctx.Timestamp,
		}
		
		if err := e.auditRepo.SaveFilterResult(audit); err != nil {
			continue
		}
	}
}

func (e *Engine) sortRulesByPriority() {
	sort.Slice(e.rules, func(i, j int) bool {
		return e.rules[i].Priority() > e.rules[j].Priority()
	})
}

func (e *Engine) GetFilterStats(ctx models.Context, tasks []models.Task) FilterStats {
	stats := FilterStats{
		TotalTasks:    len(tasks),
		VisibleTasks:  0,
		FilterResults: make(map[string]FilterRuleStats),
	}
	
	for _, rule := range e.rules {
		ruleStats := FilterRuleStats{
			Name:         rule.Name(),
			TasksVisible: 0,
			TasksHidden:  0,
			Reasons:      make(map[string]int),
		}
		
		for _, task := range tasks {
			visible, reason := rule.Apply(ctx, task)
			if visible {
				ruleStats.TasksVisible++
			} else {
				ruleStats.TasksHidden++
			}
			ruleStats.Reasons[reason]++
		}
		
		stats.FilterResults[rule.Name()] = ruleStats
	}
	
	visibleTasks, _ := e.FilterTasks(ctx, tasks)
	stats.VisibleTasks = len(visibleTasks)
	
	return stats
}

type FilterStats struct {
	TotalTasks    int                        `json:"total_tasks"`
	VisibleTasks  int                        `json:"visible_tasks"`
	FilterResults map[string]FilterRuleStats `json:"filter_results"`
}

type FilterRuleStats struct {
	Name         string         `json:"name"`
	TasksVisible int            `json:"tasks_visible"`
	TasksHidden  int            `json:"tasks_hidden"`
	Reasons      map[string]int `json:"reasons"`
}

func (e *Engine) ApplySingleFilter(filterName string, ctx models.Context, task models.Task) (bool, string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	for _, rule := range e.rules {
		if rule.Name() == filterName {
			visible, reason := rule.Apply(ctx, task)
			return visible, reason, nil
		}
	}
	
	return false, "", fmt.Errorf("filter '%s' not found", filterName)
}

func (e *Engine) GetRegisteredFilters() []FilterInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	filters := make([]FilterInfo, len(e.rules))
	for i, rule := range e.rules {
		filters[i] = FilterInfo{
			Name:     rule.Name(),
			Priority: rule.Priority(),
		}
	}
	
	return filters
}

type FilterInfo struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
}

func (e *Engine) DisableFilter(filterName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	switch filterName {
	case "location":
		e.config.EnableLocationFilter = false
	case "time":
		e.config.EnableTimeFilter = false
	case "dependency":
		e.config.EnableDependencyFilter = false
	case "priority":
		e.config.EnablePriorityFilter = false
	default:
		return fmt.Errorf("unknown filter: %s", filterName)
	}
	
	return nil
}

func (e *Engine) EnableFilter(filterName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	switch filterName {
	case "location":
		e.config.EnableLocationFilter = true
	case "time":
		e.config.EnableTimeFilter = true
	case "dependency":
		e.config.EnableDependencyFilter = true
	case "priority":
		e.config.EnablePriorityFilter = true
	default:
		return fmt.Errorf("unknown filter: %s", filterName)
	}
	
	return nil
}

func (e *Engine) GetConfig() FilterConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

func (e *Engine) UpdateConfig(config FilterConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = config
}

func generateAuditID() string {
	return fmt.Sprintf("audit_%d", time.Now().UnixNano())
}

func (e *Engine) ExplainTaskVisibility(ctx models.Context, task models.Task) TaskVisibilityExplanation {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	explanation := TaskVisibilityExplanation{
		TaskID:      task.ID,
		TaskTitle:   task.Title,
		IsVisible:   true,
		FilterResults: []FilterExplanation{},
	}
	
	for _, rule := range e.rules {
		visible, reason := rule.Apply(ctx, task)
		
		filterExpl := FilterExplanation{
			FilterName: rule.Name(),
			Passed:     visible,
			Reason:     reason,
			Priority:   rule.Priority(),
		}
		
		explanation.FilterResults = append(explanation.FilterResults, filterExpl)
		
		if !visible {
			explanation.IsVisible = false
		}
	}
	
	return explanation
}

