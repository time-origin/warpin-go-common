package database

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// GenericRepository 是一个通用的数据访问仓库。
type GenericRepository[T any] struct {
	db           *gorm.DB // 新增：保存 DB 实例
	fnCreate     func(ctx context.Context, params any) (T, error)
	fnFindByID   func(ctx context.Context, id int64) (T, error)
	fnUpdate     func(ctx context.Context, params any) (T, error)
	fnSoftDelete func(ctx context.Context, id int64) error
	fnHardDelete func(ctx context.Context, id int64) error
	fnList       func(ctx context.Context, params any) ([]T, error)
	fnCount      func(ctx context.Context, params any) (int64, error)
}

// --- 公共方法 ---

// GetDB 返回底层的 GORM DB 实例。
func (r *GenericRepository[T]) GetDB() *gorm.DB {
	return r.db
}

// Create 执行创建操作。
func (r *GenericRepository[T]) Create(ctx context.Context, params any) (T, error) {
	if r.fnCreate == nil {
		var zero T
		return zero, fmt.Errorf("Create function not provided")
	}
	return r.fnCreate(ctx, params)
}

// FindByID 根据 ID 查找实体。
func (r *GenericRepository[T]) FindByID(ctx context.Context, id int64) (T, error) {
	if r.fnFindByID == nil {
		var zero T
		return zero, fmt.Errorf("FindByID function not provided")
	}
	return r.fnFindByID(ctx, id)
}

// Update 执行更新操作。
func (r *GenericRepository[T]) Update(ctx context.Context, params any) (T, error) {
	if r.fnUpdate == nil {
		var zero T
		return zero, fmt.Errorf("Update function not provided")
	}
	return r.fnUpdate(ctx, params)
}

// Delete 方法封装了软删除和硬删除的逻辑。
func (r *GenericRepository[T]) Delete(ctx context.Context, id int64, hard bool) error {
	if hard {
		if r.fnHardDelete == nil {
			return fmt.Errorf("HardDelete function not provided")
		}
		return r.fnHardDelete(ctx, id)
	} else {
		if r.fnSoftDelete == nil {
			return fmt.Errorf("SoftDelete function not provided")
		}
		return r.fnSoftDelete(ctx, id)
	}
}

// List 并发执行列表查询和计数查询，确保查询条件一致。
func (r *GenericRepository[T]) List(ctx context.Context, params any) ([]T, int64, error) {
	if r.fnList == nil || r.fnCount == nil {
		return nil, 0, fmt.Errorf("List/Count functions not provided")
	}

	var g errgroup.Group
	var items []T
	var total int64

	g.Go(func() error {
		var err error
		items, err = r.fnList(ctx, params)
		return err
	})

	g.Go(func() error {
		var err error
		total, err = r.fnCount(ctx, params)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// --- 构造函数辅助类型 ---

// RepositoryFuncs 用于在构造 GenericRepository 时传入所有必要的函数。
type RepositoryFuncs[T any] struct {
	Create     func(ctx context.Context, params any) (T, error)
	FindByID   func(ctx context.Context, id int64) (T, error)
	Update     func(ctx context.Context, params any) (T, error)
	SoftDelete func(ctx context.Context, id int64) error
	HardDelete func(ctx context.Context, id int64) error
	List       func(ctx context.Context, params any) ([]T, error)
	Count      func(ctx context.Context, params any) (int64, error)
}

// NewGenericRepository 创建一个新的通用仓库实例。
// 现在需要传入 db 实例，以便 GetDB 方法使用。
func NewGenericRepository[T any](db *gorm.DB, funcs RepositoryFuncs[T]) *GenericRepository[T] {
	return &GenericRepository[T]{
		db:           db,
		fnCreate:     funcs.Create,
		fnFindByID:   funcs.FindByID,
		fnUpdate:     funcs.Update,
		fnSoftDelete: funcs.SoftDelete,
		fnHardDelete: funcs.HardDelete,
		fnList:       funcs.List,
		fnCount:      funcs.Count,
	}
}
