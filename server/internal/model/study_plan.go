package model

import "time"

// StudyPlan 学习计划主表
type StudyPlan struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint64     `gorm:"index:idx_user_id;not null" json:"user_id"`
	PlanName       string     `gorm:"type:varchar(128);not null" json:"plan_name"`
	ExamDate       string     `gorm:"type:date;not null" json:"exam_date"`
	StartDate      string     `gorm:"type:date;not null" json:"start_date"`
	Status         string     `gorm:"type:varchar(20);default:'active'" json:"status"`
	PlanType       string     `gorm:"type:varchar(20);default:'full'" json:"plan_type"`
	Version        int        `gorm:"default:1" json:"version"`
	TotalTasks     int        `gorm:"default:0" json:"total_tasks"`
	CompletedTasks int        `gorm:"default:0" json:"completed_tasks"`
	AIPrompt       string     `gorm:"column:ai_prompt;type:text" json:"ai_prompt,omitempty"`
	TargetSchool   string     `gorm:"type:varchar(128)" json:"target_school,omitempty"`
	TargetMajor    string     `gorm:"type:varchar(128)" json:"target_major,omitempty"`
	Materials      JSONMap    `gorm:"type:jsonb" json:"materials,omitempty"`
	PlanPhases     JSONList   `gorm:"type:jsonb" json:"plan_phases,omitempty"`
	LastRefreshAt  *time.Time `json:"last_refresh_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// 关联
	Tasks []StudyTask `gorm:"foreignKey:PlanID" json:"tasks,omitempty"`
}

func (StudyPlan) TableName() string { return "study_plans" }

// StudyTask 学习任务表
type StudyTask struct {
	ID              uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PlanID          uint64     `gorm:"index:idx_plan_id;not null" json:"plan_id"`
	UserID          uint64     `gorm:"index:idx_user_date;not null" json:"user_id"`
	TaskDate        string     `gorm:"type:date;not null" json:"task_date"`
	StartTime       string     `gorm:"type:time;not null" json:"start_time"`
	EndTime         string     `gorm:"type:time;not null" json:"end_time"`
	Content         string     `gorm:"type:text;not null" json:"content"`
	TaskType        string     `gorm:"type:varchar(20);default:'study'" json:"task_type"`
	Subject         string     `gorm:"type:varchar(20)" json:"subject,omitempty"`
	Chapter         string     `gorm:"type:varchar(128)" json:"chapter,omitempty"`
	KnowledgePoints JSONList   `gorm:"type:json" json:"knowledge_points,omitempty"`
	PomodoroCount   int        `gorm:"default:1" json:"pomodoro_count"`
	Status          string     `gorm:"type:varchar(20);default:'pending'" json:"status"`
	ActualStartTime *time.Time `json:"actual_start_time,omitempty"`
	ActualEndTime   *time.Time `json:"actual_end_time,omitempty"`
	Notes           string     `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (StudyTask) TableName() string { return "study_tasks" }
