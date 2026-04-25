package database

import "gorm.io/gorm"

type Repository[T any] interface {
	FindAll() ([]T, error)
	Where(query string, args ...interface{}) ([]T, error)
	Create(entity *T) error
	Update(entity *T) error
	Delete(id int) error
}

type ConnectedRepository[T any] struct {
	DB *gorm.DB
}

func (r *ConnectedRepository[T]) Where(query string, args ...interface{}) ([]T, error) {
	var entities []T
	result := r.DB.Where(query, args...).Find(&entities)
	return entities, result.Error
}

func (r *ConnectedRepository[T]) FindAll() ([]T, error) {
	var entities []T
	result := r.DB.Find(&entities)
	return entities, result.Error
}

func (r *ConnectedRepository[T]) Create(entity *T) error {
	result := r.DB.Create(entity)
	return result.Error
}

func (r *ConnectedRepository[T]) Update(entity *T) error {
	result := r.DB.Save(entity)
	return result.Error
}

func (r *ConnectedRepository[T]) Delete(id int) error {
	result := r.DB.Delete(new(T), id)
	return result.Error
}

func NewRepository[T any](db *gorm.DB) Repository[T] {
	return &ConnectedRepository[T]{DB: db}
}
