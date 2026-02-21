package repository

import (
	"github.com/kaoyan/server/internal/model"
	"gorm.io/gorm"
)

type PlanRepo struct {
	db *gorm.DB
}

func NewPlanRepo(db *gorm.DB) *PlanRepo {
	return &PlanRepo{db: db}
}

// Create 创建学习计划
func (r *PlanRepo) Create(plan *model.StudyPlan) error {
	return r.db.Create(plan).Error
}

// FindByID 根据ID查找计划
func (r *PlanRepo) FindByID(id uint64) (*model.StudyPlan, error) {
	var plan model.StudyPlan
	err := r.db.First(&plan, id).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// FindByIDTx 使用指定事务查找计划
func (r *PlanRepo) FindByIDTx(tx *gorm.DB, id uint64) (*model.StudyPlan, error) {
	var plan model.StudyPlan
	err := tx.First(&plan, id).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// FindByIDWithTasks 查找计划及其任务
func (r *PlanRepo) FindByIDWithTasks(id uint64) (*model.StudyPlan, error) {
	var plan model.StudyPlan
	err := r.db.Preload("Tasks", func(db *gorm.DB) *gorm.DB {
		return db.Order("task_date ASC, start_time ASC")
	}).First(&plan, id).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// ListByUserID 查询用户的计划列表
func (r *PlanRepo) ListByUserID(userID uint64, page, pageSize int) ([]model.StudyPlan, int64, error) {
	var plans []model.StudyPlan
	var total int64

	countDB := r.db.Session(&gorm.Session{})
	if err := countDB.Model(&model.StudyPlan{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	queryDB := r.db.Session(&gorm.Session{})
	err := queryDB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&plans).Error
	return plans, total, err
}

// Update 更新计划
func (r *PlanRepo) Update(plan *model.StudyPlan) error {
	return r.db.Save(plan).Error
}

// Delete 删除计划
func (r *PlanRepo) Delete(id uint64) error {
	return r.db.Delete(&model.StudyPlan{}, id).Error
}

// GetActivePlan 获取用户当前激活的计划
func (r *PlanRepo) GetActivePlan(userID uint64) (*model.StudyPlan, error) {
	var plan model.StudyPlan
	err := r.db.Where("user_id = ? AND status = 'active'", userID).
		Order("created_at DESC").
		First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// ListActive 获取所有活跃计划（用于定时刷新）
func (r *PlanRepo) ListActive() ([]model.StudyPlan, error) {
	var plans []model.StudyPlan
	err := r.db.Where("status = 'active'").Find(&plans).Error
	return plans, err
}
