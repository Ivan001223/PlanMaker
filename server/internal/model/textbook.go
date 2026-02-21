package model

import "time"

// Textbook 教材表
type Textbook struct {
	ID                   uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID               uint64    `gorm:"index:idx_user_id;not null" json:"user_id"`
	Title                string    `gorm:"type:varchar(256);not null" json:"title"`
	Subject              string    `gorm:"type:varchar(20);not null" json:"subject"`
	FileKey              string    `gorm:"type:varchar(512)" json:"file_key,omitempty"`
	FileSize             int64     `gorm:"default:0" json:"file_size"`
	ParseStatus          string    `gorm:"type:varchar(20);default:'pending'" json:"parse_status"`
	ParseTaskID          string    `gorm:"type:varchar(128)" json:"parse_task_id,omitempty"`
	TotalChapters        int       `gorm:"default:0" json:"total_chapters"`
	TotalKnowledgePoints int       `gorm:"default:0" json:"total_knowledge_points"`
	MongoDocID           string    `gorm:"type:varchar(64)" json:"mongo_doc_id,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (Textbook) TableName() string { return "textbooks" }
