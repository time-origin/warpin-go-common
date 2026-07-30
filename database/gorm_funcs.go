package database

import (
	"context"
	"fmt"
	"github.com/time-origin/warpin-go-common/query"
	"gorm.io/gorm"
	"reflect"
	"strings"
)

// applyConditions 是一个新的辅助函数，用于智能地构建查询。
func applyConditions(tx *gorm.DB, params any) *gorm.DB {
	val := reflect.ValueOf(params)
	typ := val.Type()

	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = val.Type()
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过 BaseCondition 结构体本身
		if fieldType.Type == reflect.TypeOf(condition.BaseCondition{}) {
			continue
		}

		// 跳过零值字段
		if field.IsZero() {
			continue
		}

		dbTag := fieldType.Tag.Get("db")
		dbColumnName := strings.Split(dbTag, ",")[0]
		if dbColumnName == "" {
			jsonTag := fieldType.Tag.Get("json")
			dbColumnName = strings.Split(jsonTag, ",")[0]
		}
		if dbColumnName == "" || dbColumnName == "-" {
			continue
		}

		var value interface{}
		if field.Kind() == reflect.Ptr {
			value = field.Elem().Interface()
		} else {
			value = field.Interface()
		}

		queryMode := fieldType.Tag.Get("query")
		if queryMode == "gte" {
			tx = tx.Where(fmt.Sprintf(`"%s" >= ?`, dbColumnName), value)
			continue
		}
		if queryMode == "lte" {
			tx = tx.Where(fmt.Sprintf(`"%s" <= ?`, dbColumnName), value)
			continue
		}
		// CSV-backed IN filters stay opt-in so ordinary text fields containing
		// commas retain their existing fuzzy-match semantics.
		if str, ok := value.(string); ok {
			if queryMode == "csv_in" {
				parts := strings.Split(str, ",")
				values := make([]string, 0, len(parts))
				for _, part := range parts {
					if trimmed := strings.TrimSpace(part); trimmed != "" {
						values = append(values, trimmed)
					}
				}
				if len(values) > 0 {
					tx = tx.Where(fmt.Sprintf(`"%s" IN ?`, dbColumnName), values)
				}
			} else {
				tx = tx.Where(fmt.Sprintf(`"%s" LIKE ?`, dbColumnName), "%"+str+"%")
			}
		} else {
			tx = tx.Where(fmt.Sprintf(`"%s" = ?`, dbColumnName), value)
		}
	}
	return tx
}

// NewGormFuncs 创建并返回一个 RepositoryFuncs 结构体。
func NewGormFuncs[T any](db *gorm.DB) RepositoryFuncs[T] {
	return RepositoryFuncs[T]{
		Create: func(ctx context.Context, entity any) (T, error) {
			var result T
			e, ok := entity.(*T)
			if !ok {
				return result, gorm.ErrInvalidData
			}
			err := db.WithContext(ctx).Create(e).Error
			return *e, err
		},
		FindByID: func(ctx context.Context, id int64) (T, error) {
			var result T
			err := db.WithContext(ctx).First(&result, id).Error
			return result, err
		},
		Update: func(ctx context.Context, entity any) (T, error) {
			var result T
			e, ok := entity.(*T)
			if !ok {
				return result, gorm.ErrInvalidData
			}
			err := db.WithContext(ctx).Save(e).Error
			return *e, err
		},
		SoftDelete: func(ctx context.Context, id int64) error {
			var entity T
			return db.WithContext(ctx).Delete(&entity, id).Error
		},
		HardDelete: func(ctx context.Context, id int64) error {
			var entity T
			return db.WithContext(ctx).Unscoped().Delete(&entity, id).Error
		},
		List: func(ctx context.Context, params any) ([]T, error) {
			var items []T
			tx := db.WithContext(ctx).Model(new(T))

			// 应用过滤条件
			tx = applyConditions(tx, params)

			// 提取并应用分页和排序
			v := reflect.ValueOf(params)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			baseCondField := v.FieldByName("BaseCondition")
			if baseCondField.IsValid() {
				if bc, ok := baseCondField.Interface().(condition.BaseCondition); ok {
					limit := bc.GetLimit()
					offset := (bc.GetPage() - 1) * limit
					tx = tx.Limit(limit).Offset(offset)
					if orderBy := bc.GetOrderBy(); orderBy != "" {
						tx = tx.Order(orderBy)
					}
				}
			}

			err := tx.Find(&items).Error
			return items, err
		},
		Count: func(ctx context.Context, params any) (int64, error) {
			var count int64
			tx := db.WithContext(ctx).Model(new(T))

			// 应用过滤条件
			tx = applyConditions(tx, params)

			err := tx.Count(&count).Error
			return count, err
		},
	}
}
