package repository

import (
	"github.com/kaoyan/server/internal/model"
	"gorm.io/gorm"
)

type ChatRepo struct {
	db *gorm.DB
}

func NewChatRepo(db *gorm.DB) *ChatRepo {
	return &ChatRepo{db: db}
}

func (r *ChatRepo) CreateSession(session *model.ChatSession) error {
	return r.db.Create(session).Error
}

func (r *ChatRepo) FindSessionByID(id uint64) (*model.ChatSession, error) {
	var session model.ChatSession
	err := r.db.Preload("Messages", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).First(&session, id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *ChatRepo) ListSessionsByUserID(userID uint64) ([]model.ChatSession, error) {
	var sessions []model.ChatSession
	err := r.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *ChatRepo) CreateMessage(msg *model.ChatMessage) error {
	return r.db.Create(msg).Error
}

func (r *ChatRepo) ListRecentMessages(sessionID uint64, limit int) ([]model.ChatMessage, error) {
	var messages []model.ChatMessage
	r.db.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}
