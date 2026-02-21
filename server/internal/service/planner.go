package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kaoyan/server/internal/model"
	"github.com/kaoyan/server/internal/pkg/llm"
	"github.com/kaoyan/server/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PlannerService struct {
	planRepo *repository.PlanRepo
	taskRepo *repository.TaskRepo
	userRepo *repository.UserRepo
	aiClient *llm.Client
	aiModel  string
	logger   *zap.Logger
}

func NewPlannerService(
	planRepo *repository.PlanRepo,
	taskRepo *repository.TaskRepo,
	userRepo *repository.UserRepo,
	aiClient *llm.Client,
	aiModel string,
	logger *zap.Logger,
) *PlannerService {
	return &PlannerService{
		planRepo: planRepo,
		taskRepo: taskRepo,
		userRepo: userRepo,
		aiClient: aiClient,
		aiModel:  aiModel,
		logger:   logger,
	}
}

// ExamSubject 考试科目详情
type ExamSubject struct {
	Category string `json:"category"` // math/english/politics/professional
	Name     string `json:"name"`     // 数学一/英语二/408计算机综合
}

// GeneratePlanRequest 生成计划请求
type GeneratePlanRequest struct {
	UserID          uint64            `json:"user_id"`
	PlanName        string            `json:"plan_name"`
	ExamDate        string            `json:"exam_date"`
	Subjects        []string          `json:"subjects"`
	DailyHours      float64           `json:"daily_hours"`
	WeakSubjects    []string          `json:"weak_subjects"`
	StartTime       string            `json:"start_time"`       // "08:00"
	EndTime         string            `json:"end_time"`         // "22:00"
	RestDays        []int             `json:"rest_days"`        // [0,6] = 周日,周六
	PomodoroMinutes int               `json:"pomodoro_minutes"` // 番茄钟时长
	BreakMinutes    int               `json:"break_minutes"`    // 休息时长
	TargetSchool    string            `json:"target_school"`    // 目标院校
	TargetMajor     string            `json:"target_major"`     // 目标专业
	Materials       map[string]string `json:"materials"`        // 科目→教辅材料
	ExamSubjects    []ExamSubject     `json:"exam_subjects"`    // 具体考试科目
}

// GeneratedTask AI 生成的任务
type GeneratedTask struct {
	Date      string `json:"date"`       // "2026-03-01"
	StartTime string `json:"start_time"` // "08:00"
	EndTime   string `json:"end_time"`   // "08:25"
	Subject   string `json:"subject"`
	Content   string `json:"content"`
	TaskType  string `json:"task_type"`
	Chapter   string `json:"chapter,omitempty"`
}

// GeneratePlan 生成学习计划
func (s *PlannerService) GeneratePlan(req *GeneratePlanRequest) (*model.StudyPlan, error) {
	// 1. 构建 AI prompt
	prompt := s.buildPlanningPrompt(req)

	// 2. 调用 AI 生成计划
	aiResp, err := s.aiClient.ChatCompletion(llm.ChatRequest{
		Model: s.aiModel,
		Messages: []llm.Message{
			{Role: "system", Content: plannerSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		MaxTokens:   8192,
	})
	if err != nil {
		s.logger.Error("AI生成计划失败", zap.Error(err))
		return nil, fmt.Errorf("AI生成计划失败: %w", err)
	}

	// 3. 解析 AI 响应
	var tasks []GeneratedTask
	var phases []map[string]interface{}

	if len(aiResp.Choices) == 0 {
		s.logger.Warn("AI返回空Choices，使用规则引擎降级")
		tasks = s.generateByRules(req)
		phases = s.generateDefaultPhases(req)
	} else {
		aiContent := aiResp.Choices[0].Message.Content
		tasks, phases, err = s.parseAIResponse(aiContent, req)
		if err != nil {
			s.logger.Warn("AI响应解析失败,使用规则引擎生成", zap.Error(err))
			tasks = s.generateByRules(req)
			phases = s.generateDefaultPhases(req)
		}
	}

	// 4. 构建材料 JSONMap
	var materials model.JSONMap
	if req.Materials != nil {
		matJSON, _ := json.Marshal(req.Materials)
		_ = json.Unmarshal(matJSON, &materials)
	}

	// 5. 构建阶段 JSONList
	var planPhases model.JSONList
	if len(phases) > 0 {
		phaseJSON, _ := json.Marshal(phases)
		_ = json.Unmarshal(phaseJSON, &planPhases)
	}

	// 6. 创建计划记录
	plan := &model.StudyPlan{
		UserID:       req.UserID,
		PlanName:     req.PlanName,
		ExamDate:     req.ExamDate,
		StartDate:    time.Now().Format("2006-01-02"),
		Status:       "active",
		PlanType:     "full",
		TotalTasks:   len(tasks),
		AIPrompt:     prompt,
		TargetSchool: req.TargetSchool,
		TargetMajor:  req.TargetMajor,
		Materials:    materials,
		PlanPhases:   planPhases,
	}

	if err := s.planRepo.Create(plan); err != nil {
		return nil, fmt.Errorf("创建计划失败: %w", err)
	}

	// 7. 创建任务记录
	var studyTasks []model.StudyTask
	for _, t := range tasks {
		studyTasks = append(studyTasks, model.StudyTask{
			PlanID:    plan.ID,
			UserID:    req.UserID,
			TaskDate:  t.Date,
			StartTime: t.StartTime,
			EndTime:   t.EndTime,
			Content:   t.Content,
			TaskType:  t.TaskType,
			Subject:   t.Subject,
			Chapter:   t.Chapter,
			Status:    "pending",
		})
	}

	if len(studyTasks) > 0 {
		if err := s.taskRepo.BatchCreate(studyTasks); err != nil {
			return nil, fmt.Errorf("创建任务失败: %w", err)
		}
	}

	return plan, nil
}

// RefreshPlan 根据完成情况刷新计划
func (s *PlannerService) RefreshPlan(planID uint64) (*model.StudyPlan, error) {
	plan, err := s.planRepo.FindByIDWithTasks(planID)
	if err != nil {
		return nil, fmt.Errorf("计划不存在: %w", err)
	}

	if plan.Status != "active" {
		return nil, fmt.Errorf("计划非活跃状态，无法刷新")
	}

	// 统计完成情况
	total, completed, _ := s.taskRepo.GetCompletionStats(planID)
	today := time.Now().Format("2006-01-02")

	// 查找未完成的过期任务（昨天及之前）
	pendingTasks, _ := s.taskRepo.ListPendingBefore(plan.UserID, planID, today)

	// 构建刷新 prompt
	prompt := s.buildRefreshPrompt(plan, int(total), int(completed), pendingTasks)

	aiResp, err := s.aiClient.ChatCompletion(llm.ChatRequest{
		Model: s.aiModel,
		Messages: []llm.Message{
			{Role: "system", Content: plannerSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		MaxTokens:   8192,
	})
	if err != nil {
		s.logger.Error("AI刷新计划失败", zap.Error(err))
		return nil, fmt.Errorf("AI刷新计划失败: %w", err)
	}

	if len(aiResp.Choices) == 0 {
		s.logger.Warn("AI刷新返回空Choices")
		return plan, nil
	}

	aiContent := aiResp.Choices[0].Message.Content

	newTasks, _, err := s.parseAIResponse(aiContent, &GeneratePlanRequest{
		UserID:   plan.UserID,
		ExamDate: plan.ExamDate,
	})
	if err != nil {
		s.logger.Warn("刷新解析失败", zap.Error(err))
		return plan, nil
	}

	// 删除未来未开始的任务(仅 pending 状态)
	if err := s.taskRepo.DeleteFuturePending(planID, today); err != nil {
		s.logger.Error("删除未来任务失败", zap.Error(err))
	}

	// 将未完成的过期任务标记为 skipped
	for _, t := range pendingTasks {
		t.Status = "skipped"
		_ = s.taskRepo.Update(&t)
	}

	// 添加新生成的任务
	var studyTasks []model.StudyTask
	for _, t := range newTasks {
		studyTasks = append(studyTasks, model.StudyTask{
			PlanID:    plan.ID,
			UserID:    plan.UserID,
			TaskDate:  t.Date,
			StartTime: t.StartTime,
			EndTime:   t.EndTime,
			Content:   t.Content,
			TaskType:  t.TaskType,
			Subject:   t.Subject,
			Chapter:   t.Chapter,
			Status:    "pending",
		})
	}

	if len(studyTasks) > 0 {
		if err := s.taskRepo.BatchCreate(studyTasks); err != nil {
			s.logger.Error("创建刷新任务失败", zap.Error(err))
		}
	}

	// 更新计划统计
	now := time.Now()
	plan.LastRefreshAt = &now
	plan.Version++
	newTotal, newCompleted, _ := s.taskRepo.GetCompletionStats(planID)
	plan.TotalTasks = int(newTotal)
	plan.CompletedTasks = int(newCompleted)
	_ = s.planRepo.Update(plan)

	return plan, nil
}

// GetPlan 获取计划详情
func (s *PlannerService) GetPlan(planID uint64) (*model.StudyPlan, error) {
	return s.planRepo.FindByIDWithTasks(planID)
}

// ListPlans 获取用户计划列表
func (s *PlannerService) ListPlans(userID uint64, page, pageSize int) ([]model.StudyPlan, int64, error) {
	return s.planRepo.ListByUserID(userID, page, pageSize)
}

// GetTodayTasks 获取今日任务
func (s *PlannerService) GetTodayTasks(userID uint64) ([]model.StudyTask, error) {
	today := time.Now().Format("2006-01-02")
	return s.taskRepo.ListByDate(userID, today)
}

// GetWeekTasks 获取本周任务
func (s *PlannerService) GetWeekTasks(userID uint64) ([]model.StudyTask, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -(weekday - 1))
	endOfWeek := startOfWeek.AddDate(0, 0, 6)
	return s.taskRepo.ListByDateRange(userID, startOfWeek.Format("2006-01-02"), endOfWeek.Format("2006-01-02"))
}

// UpdateTaskStatus 更新任务状态
func (s *PlannerService) UpdateTaskStatus(taskID uint64, status string, userID uint64) error {
	// 使用事务确保数据一致性
	return s.taskRepo.DB().Transaction(func(tx *gorm.DB) error {
		task, err := s.taskRepo.FindByIDTx(tx, taskID)
		if err != nil {
			return err
		}

		// IDOR 防护：验证任务归属
		if task.UserID != userID {
			return fmt.Errorf("无权操作此任务")
		}

		task.Status = status
		if status == "in_progress" {
			now := time.Now()
			task.ActualStartTime = &now
		} else if status == "completed" {
			now := time.Now()
			task.ActualEndTime = &now
		}

		// 在事务中更新任务
		if err := tx.Save(task).Error; err != nil {
			return err
		}

		// 在事务中更新计划统计
		var total, completed int64
		tx.Model(&model.StudyTask{}).Where("plan_id = ?", task.PlanID).Count(&total)
		tx.Model(&model.StudyTask{}).Where("plan_id = ? AND status = ?", task.PlanID, "completed").Count(&completed)

		plan, err := s.planRepo.FindByIDTx(tx, task.PlanID)
		if err == nil && plan != nil {
			plan.TotalTasks = int(total)
			plan.CompletedTasks = int(completed)
			if err := tx.Save(plan).Error; err != nil {
				s.logger.Warn("更新计划统计失败", zap.Error(err))
			}
		}

		return nil
	})
}

// GetActivePlans 获取所有活跃计划（用于定时刷新）
func (s *PlannerService) GetActivePlans() ([]model.StudyPlan, error) {
	return s.planRepo.ListActive()
}

// ============================================================
// System Prompt & Prompt Builders
// ============================================================

const plannerSystemPrompt = `你是一个专业的考研学习规划引擎。你的任务是根据用户的备考信息生成详细的、覆盖整个备考周期的学习计划。

## 输出格式

请严格按照以下JSON格式返回：
{
  "phases": [
    {
      "name": "基础阶段",
      "start_date": "2026-03-01",
      "end_date": "2026-06-30",
      "focus": "夯实基础,系统学习各科教材",
      "materials": ["张宇高数18讲", "朱伟恋练有词"]
    }
  ],
  "tasks": [
    {
      "date": "2026-03-01",
      "start_time": "08:00",
      "end_time": "08:25",
      "subject": "math",
      "content": "张宇高数18讲-第一章函数与极限-1.1映射与函数",
      "task_type": "study",
      "chapter": "第一章 函数与极限"
    }
  ]
}

## 生成规则

1. **阶段划分**（根据距考试时间自动调整）：
   - 基础阶段(3-5个月)：系统学习教材和基础讲义，建立知识体系
   - 强化阶段(2-3个月)：专题突破、重难点强化、大量刷题
   - 冲刺阶段(1-2个月)：真题训练、模拟考试、查漏补缺
   - 考前阶段(2-4周)：背诵记忆、政治冲刺(肖四肖八)、心态调整

2. **任务内容必须结合教辅材料**：
   - 内容字段必须明确到教材章节，如"张宇高数18讲-第三章-微分中值定理"
   - 不要生成模糊的"复习数学"，要具体到教材+章节+知识点

3. **科目时间分配**：
   - subject可选值: math, english, politics, professional
   - task_type可选值: study(学习), review(复习), exercise(刷题), mock_exam(模考), rest(休息)
   - 薄弱科目分配更多时间（增加30-50%）
   - 每科每天至少25分钟（一个番茄钟）

4. **番茄工作法**：按番茄钟排列（默认25min学习+5min休息）

5. **任务数量**：只生成未来7天的详细每日任务（tasks），但phases要覆盖整个备考周期

6. **政治特殊规则**：
   - 距考试>6个月：每天只需30分钟
   - 距考试3-6个月：每天1小时
   - 距考试<3个月：每天2小时（冲刺背诵）`

func (s *PlannerService) buildPlanningPrompt(req *GeneratePlanRequest) string {
	var sb strings.Builder
	sb.WriteString("请为我生成考研学习计划：\n\n")

	// 基本信息
	sb.WriteString(fmt.Sprintf("考试日期: %s\n", req.ExamDate))
	sb.WriteString(fmt.Sprintf("开始日期: %s (今天)\n", time.Now().Format("2006-01-02")))

	if req.TargetSchool != "" {
		sb.WriteString(fmt.Sprintf("目标院校: %s\n", req.TargetSchool))
	}
	if req.TargetMajor != "" {
		sb.WriteString(fmt.Sprintf("目标专业: %s\n", req.TargetMajor))
	}

	// 考试科目
	if len(req.ExamSubjects) > 0 {
		sb.WriteString("考试科目:\n")
		for _, subj := range req.ExamSubjects {
			sb.WriteString(fmt.Sprintf("  - %s (%s)\n", subj.Name, subj.Category))
		}
	} else if len(req.Subjects) > 0 {
		sb.WriteString(fmt.Sprintf("备考科目: %v\n", req.Subjects))
	}

	// 教辅材料
	if len(req.Materials) > 0 {
		sb.WriteString("使用教辅材料:\n")
		for subj, mat := range req.Materials {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", subjectName(subj), mat))
		}
	}

	// 学习习惯
	sb.WriteString(fmt.Sprintf("每日学习时长: %.1f小时\n", req.DailyHours))
	if len(req.WeakSubjects) > 0 {
		sb.WriteString(fmt.Sprintf("薄弱科目: %v\n", req.WeakSubjects))
	}
	sb.WriteString(fmt.Sprintf("学习时间段: %s - %s\n", req.StartTime, req.EndTime))

	pomoDuration := req.PomodoroMinutes
	if pomoDuration == 0 {
		pomoDuration = 25
	}
	breakDuration := req.BreakMinutes
	if breakDuration == 0 {
		breakDuration = 5
	}
	sb.WriteString(fmt.Sprintf("番茄钟时长: %d分钟\n", pomoDuration))
	sb.WriteString(fmt.Sprintf("休息时长: %d分钟\n", breakDuration))

	if len(req.RestDays) > 0 {
		sb.WriteString(fmt.Sprintf("休息日: %v (0=周日,6=周六)\n", req.RestDays))
	}

	sb.WriteString("\n请生成覆盖整个备考周期的阶段规划(phases)，以及未来7天的详细每日学习计划(tasks)。\n")
	sb.WriteString("任务内容必须结合教辅材料具体到章节。")

	return sb.String()
}

func (s *PlannerService) buildRefreshPrompt(plan *model.StudyPlan, total, completed int, pendingTasks []model.StudyTask) string {
	var sb strings.Builder
	sb.WriteString("请根据以下学习进度刷新考研学习计划：\n\n")

	sb.WriteString(fmt.Sprintf("计划名称: %s\n", plan.PlanName))
	sb.WriteString(fmt.Sprintf("考试日期: %s\n", plan.ExamDate))
	sb.WriteString(fmt.Sprintf("今天日期: %s\n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("总任务数: %d, 已完成: %d, 完成率: %.1f%%\n",
		total, completed, float64(completed)/float64(max(total, 1))*100))

	if plan.TargetSchool != "" {
		sb.WriteString(fmt.Sprintf("目标院校: %s\n", plan.TargetSchool))
	}
	if plan.TargetMajor != "" {
		sb.WriteString(fmt.Sprintf("目标专业: %s\n", plan.TargetMajor))
	}

	// 教辅材料
	if plan.Materials != nil {
		sb.WriteString("使用教辅材料:\n")
		for subj, mat := range plan.Materials {
			sb.WriteString(fmt.Sprintf("  - %s: %v\n", subjectName(subj), mat))
		}
	}

	// 未完成的过期任务
	if len(pendingTasks) > 0 {
		sb.WriteString(fmt.Sprintf("\n以下 %d 个任务未完成，需要顺延到后续计划中：\n", len(pendingTasks)))
		for _, t := range pendingTasks {
			sb.WriteString(fmt.Sprintf("  - [%s] %s (%s %s-%s)\n", subjectName(t.Subject), t.Content, t.TaskDate, t.StartTime, t.EndTime))
		}
	}

	sb.WriteString("\n请生成未来7天的新任务，将未完成的内容优先安排，并根据进度适当调整节奏。\n")
	sb.WriteString("任务内容必须结合教辅材料具体到章节。")

	return sb.String()
}

// parseAIResponse 解析AI响应
func (s *PlannerService) parseAIResponse(content string, req *GeneratePlanRequest) ([]GeneratedTask, []map[string]interface{}, error) {
	// 尝试从 markdown code block 中提取 JSON
	jsonStr := extractJSON(content)

	var result struct {
		Phases []map[string]interface{} `json:"phases"`
		Tasks  []GeneratedTask          `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, nil, fmt.Errorf("解析AI响应JSON失败: %w", err)
	}
	if len(result.Tasks) == 0 {
		return nil, nil, fmt.Errorf("AI返回空任务列表")
	}
	return result.Tasks, result.Phases, nil
}

// extractJSON 从可能包含 markdown code block 的内容中提取 JSON
func extractJSON(content string) string {
	// 尝试提取 ```json ... ``` 块
	if idx := strings.Index(content, "```json"); idx != -1 {
		start := idx + 7
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}
	// 尝试提取 ``` ... ``` 块
	if idx := strings.Index(content, "```"); idx != -1 {
		start := idx + 3
		// 跳过可能的语言标识符行
		if nlIdx := strings.Index(content[start:], "\n"); nlIdx != -1 {
			start += nlIdx + 1
		}
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}
	// 尝试找到第一个 { 和最后一个 }
	firstBrace := strings.Index(content, "{")
	lastBrace := strings.LastIndex(content, "}")
	if firstBrace != -1 && lastBrace > firstBrace {
		return content[firstBrace : lastBrace+1]
	}
	return content
}

// generateDefaultPhases 生成默认阶段
func (s *PlannerService) generateDefaultPhases(req *GeneratePlanRequest) []map[string]interface{} {
	examDate, _ := time.Parse("2006-01-02", req.ExamDate)
	now := time.Now()
	totalDays := int(examDate.Sub(now).Hours() / 24)

	var phases []map[string]interface{}

	if totalDays > 180 { // > 6 months: 4 phases
		phase1End := now.AddDate(0, 0, totalDays*40/100)
		phase2End := now.AddDate(0, 0, totalDays*70/100)
		phase3End := now.AddDate(0, 0, totalDays*90/100)
		phases = []map[string]interface{}{
			{"name": "基础阶段", "start_date": now.Format("2006-01-02"), "end_date": phase1End.Format("2006-01-02"), "focus": "系统学习教材，建立知识体系"},
			{"name": "强化阶段", "start_date": phase1End.AddDate(0, 0, 1).Format("2006-01-02"), "end_date": phase2End.Format("2006-01-02"), "focus": "专题突破，大量刷题"},
			{"name": "冲刺阶段", "start_date": phase2End.AddDate(0, 0, 1).Format("2006-01-02"), "end_date": phase3End.Format("2006-01-02"), "focus": "真题训练，模拟考试"},
			{"name": "考前阶段", "start_date": phase3End.AddDate(0, 0, 1).Format("2006-01-02"), "end_date": req.ExamDate, "focus": "查漏补缺，背诵冲刺"},
		}
	} else if totalDays > 90 { // 3-6 months: 3 phases
		phase1End := now.AddDate(0, 0, totalDays*50/100)
		phase2End := now.AddDate(0, 0, totalDays*85/100)
		phases = []map[string]interface{}{
			{"name": "基础强化阶段", "start_date": now.Format("2006-01-02"), "end_date": phase1End.Format("2006-01-02"), "focus": "快速过基础+重点强化"},
			{"name": "冲刺阶段", "start_date": phase1End.AddDate(0, 0, 1).Format("2006-01-02"), "end_date": phase2End.Format("2006-01-02"), "focus": "真题模拟+查漏补缺"},
			{"name": "考前阶段", "start_date": phase2End.AddDate(0, 0, 1).Format("2006-01-02"), "end_date": req.ExamDate, "focus": "背诵冲刺+心态调整"},
		}
	} else { // < 3 months: 2 phases
		phase1End := now.AddDate(0, 0, totalDays*70/100)
		phases = []map[string]interface{}{
			{"name": "密集冲刺", "start_date": now.Format("2006-01-02"), "end_date": phase1End.Format("2006-01-02"), "focus": "真题+重点+薄弱项突破"},
			{"name": "考前冲刺", "start_date": phase1End.AddDate(0, 0, 1).Format("2006-01-02"), "end_date": req.ExamDate, "focus": "模考+背诵+心态调整"},
		}
	}

	return phases
}

// generateByRules 规则引擎降级生成
func (s *PlannerService) generateByRules(req *GeneratePlanRequest) []GeneratedTask {
	var tasks []GeneratedTask
	subjects := req.Subjects
	if len(subjects) == 0 {
		subjects = []string{"math", "english", "politics", "professional"}
	}

	pomoDuration := req.PomodoroMinutes
	if pomoDuration == 0 {
		pomoDuration = 25
	}
	breakDuration := req.BreakMinutes
	if breakDuration == 0 {
		breakDuration = 5
	}

	// 生成未来7天的计划
	for day := 0; day < 7; day++ {
		date := time.Now().AddDate(0, 0, day)
		weekday := int(date.Weekday())

		// 跳过休息日
		isRest := false
		for _, rd := range req.RestDays {
			if rd == weekday {
				isRest = true
				break
			}
		}
		if isRest {
			continue
		}

		dateStr := date.Format("2006-01-02")
		startHour := 8
		startMinute := 0

		if req.StartTime != "" {
			if parsed, parseErr := time.Parse("15:04", req.StartTime); parseErr == nil {
				startHour = parsed.Hour()
				startMinute = parsed.Minute()
			}
		}

		hour := startHour
		minute := startMinute

		endHour := 22
		if req.EndTime != "" {
			if parsed, parseErr := time.Parse("15:04", req.EndTime); parseErr == nil {
				endHour = parsed.Hour()
			}
		}

		// 每个科目分配番茄钟
		for _, subject := range subjects {
			pomos := 2
			for _, ws := range req.WeakSubjects {
				if ws == subject {
					pomos = 3
					break
				}
			}

			// 生成内容：如果有教辅材料，引用教辅
			contentPrefix := subjectName(subject) + " 复习"
			if mat, ok := req.Materials[subject]; ok {
				contentPrefix = mat + " - " + subjectName(subject)
			}

			for p := 0; p < pomos; p++ {
				startStr := fmt.Sprintf("%02d:%02d", hour, minute)
				minute += pomoDuration
				if minute >= 60 {
					hour += minute / 60
					minute = minute % 60
				}
				endStr := fmt.Sprintf("%02d:%02d", hour, minute)

				tasks = append(tasks, GeneratedTask{
					Date:      dateStr,
					StartTime: startStr,
					EndTime:   endStr,
					Subject:   subject,
					Content:   fmt.Sprintf("%s - 第%d节", contentPrefix, p+1),
					TaskType:  "study",
				})

				// 休息
				breakStart := endStr
				minute += breakDuration
				if minute >= 60 {
					hour += minute / 60
					minute = minute % 60
				}
				breakEnd := fmt.Sprintf("%02d:%02d", hour, minute)

				tasks = append(tasks, GeneratedTask{
					Date:      dateStr,
					StartTime: breakStart,
					EndTime:   breakEnd,
					Subject:   "",
					Content:   "休息",
					TaskType:  "rest",
				})

				if hour >= endHour {
					break
				}
			}
			if hour >= endHour {
				break
			}
		}
	}
	return tasks
}

func subjectName(subject string) string {
	switch subject {
	case "math":
		return "数学"
	case "english":
		return "英语"
	case "politics":
		return "政治"
	case "professional":
		return "专业课"
	default:
		return subject
	}
}
