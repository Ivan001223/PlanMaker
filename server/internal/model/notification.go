package model

import "time"

// Notification 通知记录表
type Notification struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        uint64     `gorm:"index:idx_user_id;not null" json:"user_id"`
	NotifyType    string     `gorm:"type:varchar(20);not null" json:"notify_type"`
	Title         string     `gorm:"type:varchar(128);not null" json:"title"`
	Content       string     `gorm:"type:varchar(512);not null" json:"content"`
	ScheduledAt   time.Time  `gorm:"index:idx_scheduled_at;not null" json:"scheduled_at"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	Status        string     `gorm:"type:varchar(20);default:'pending';index:idx_status" json:"status"`
	RelatedTaskID *uint64    `json:"related_task_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }
