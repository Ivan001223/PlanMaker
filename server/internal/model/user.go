package model

import "time"

// User 用户表
type User struct {
	ID                 uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	OpenID             string     `gorm:"column:openid;type:varchar(64);uniqueIndex:uk_openid;not null" json:"-"`
	UnionID            string     `gorm:"column:union_id;type:varchar(64);index:idx_union_id" json:"-"`
	Nickname           string     `gorm:"type:varchar(64)" json:"nickname"`
	AvatarURL          string     `gorm:"type:varchar(512)" json:"avatar_url"`
	Phone              string     `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Membership         string     `gorm:"type:varchar(20);default:'free'" json:"membership"`
	MembershipExpireAt *time.Time `json:"membership_expire_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (User) TableName() string { return "users" }

// UserPreference 用户偏好设置
type UserPreference struct {
	ID                uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            uint64    `gorm:"uniqueIndex:uk_user_id;not null" json:"user_id"`
	ExamDate          *string   `gorm:"type:date" json:"exam_date,omitempty"`
	TargetScore       *int      `json:"target_score,omitempty"`
	DailyStudyHours   float64   `gorm:"type:decimal(3,1);default:8.0" json:"daily_study_hours"`
	WeakSubjects      JSONList  `gorm:"type:json" json:"weak_subjects,omitempty"`
	StudyStartTime    string    `gorm:"type:time;default:'08:00:00'" json:"study_start_time"`
	StudyEndTime      string    `gorm:"type:time;default:'22:00:00'" json:"study_end_time"`
	RestDays          JSONList  `gorm:"type:json" json:"rest_days,omitempty"`
	PomodoroDuration  int       `gorm:"default:25" json:"pomodoro_duration"`
	BreakDuration     int       `gorm:"default:5" json:"break_duration"`
	LongBreakDuration int       `gorm:"default:15" json:"long_break_duration"`
	LongBreakInterval int       `gorm:"default:4" json:"long_break_interval"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (UserPreference) TableName() string { return "user_preferences" }
