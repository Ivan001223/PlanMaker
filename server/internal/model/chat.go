package model

import "time"

// ChatSession 对话会话表
type ChatSession struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint64    `gorm:"index:idx_user_id;not null" json:"user_id"`
	SessionType string    `gorm:"type:varchar(20);default:'planning'" json:"session_type"`
	Title       string    `gorm:"type:varchar(128)" json:"title"`
	Status      string    `gorm:"type:varchar(20);default:'active'" json:"status"`
	Context     JSONMap   `gorm:"type:json" json:"context,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联
	Messages []ChatMessage `gorm:"foreignKey:SessionID" json:"messages,omitempty"`
}

func (ChatSession) TableName() string { return "chat_sessions" }

// ChatMessage 对话消息表
type ChatMessage struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID  uint64    `gorm:"index:idx_session_id;not null" json:"session_id"`
	Role       string    `gorm:"type:varchar(20);not null" json:"role"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	TokensUsed int       `gorm:"default:0" json:"tokens_used"`
	CreatedAt  time.Time `json:"created_at"`
}

func (ChatMessage) TableName() string { return "chat_messages" }
