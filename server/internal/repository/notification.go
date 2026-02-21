package repository

import (
	"time"

	"github.com/kaoyan/server/internal/model"
	"gorm.io/gorm"
)

type NotificationRepo struct {
	db *gorm.DB
}

func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Create(notification *model.Notification) error {
	return r.db.Create(notification).Error
}

func (r *NotificationRepo) FindPending(limit int) ([]model.Notification, error) {
	var notifications []model.Notification
	err := r.db.Where("status = 'pending' AND scheduled_at <= ?", time.Now()).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&notifications).Error
	return notifications, err
}

func (r *NotificationRepo) UpdateStatus(id uint64, status string, sentAt *time.Time) error {
	updates := map[string]interface{}{"status": status}
	if sentAt != nil {
		updates["sent_at"] = sentAt
	}
	return r.db.Model(&model.Notification{}).Where("id = ?", id).Updates(updates).Error
}

func (r *NotificationRepo) ListByUserID(userID uint64, page, pageSize int) ([]model.Notification, int64, error) {
	var notifications []model.Notification
	var total int64

	r.db.Model(&model.Notification{}).Where("user_id = ?", userID).Count(&total)

	err := r.db.Where("user_id = ?", userID).
		Order("scheduled_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&notifications).Error
	return notifications, total, err
}

func (r *NotificationRepo) FindPlanByID(planID uint64) (*model.StudyPlan, error) {
	var plan model.StudyPlan
	err := r.db.First(&plan, planID).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *NotificationRepo) FindPendingTasksByPlanID(planID uint64) ([]model.StudyTask, error) {
	var tasks []model.StudyTask
	err := r.db.Where("plan_id = ? AND task_type = 'study' AND status = 'pending'", planID).Find(&tasks).Error
	return tasks, err
}
