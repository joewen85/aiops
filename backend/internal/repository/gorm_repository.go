package repository

import (
	"context"

	"gorm.io/gorm"

	"devops-system/backend/internal/pagination"
)

type GormRepository[T any] struct {
	db *gorm.DB
}

func NewGormRepository[T any](db *gorm.DB) *GormRepository[T] {
	return &GormRepository[T]{db: db}
}

func (r *GormRepository[T]) DB() *gorm.DB {
	return r.db
}

func (r *GormRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *GormRepository[T]) GetByID(ctx context.Context, id uint) (*T, error) {
	var entity T
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *GormRepository[T]) Update(ctx context.Context, id uint, updates map[string]interface{}) (*T, error) {
	var entity T
	if err := r.db.WithContext(ctx).Model(&entity).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *GormRepository[T]) Delete(ctx context.Context, id uint) error {
	var entity T
	return r.db.WithContext(ctx).Delete(&entity, id).Error
}

func (r *GormRepository[T]) List(ctx context.Context, page pagination.Query) ([]T, int64, error) {
	var (
		entities []T
		total    int64
	)
	model := new(T)
	if err := r.db.WithContext(ctx).Model(model).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).
		Limit(page.PageSize).
		Offset(pagination.Offset(page)).
		Order("id desc").
		Find(&entities).Error; err != nil {
		return nil, 0, err
	}
	return entities, total, nil
}
