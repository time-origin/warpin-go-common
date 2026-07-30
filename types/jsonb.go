package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONB 是一种自定义类型，用于在 GORM 中处理 jsonb 字段
type JSONB json.RawMessage

// Scan 实现了 sql.Scanner 接口，允许从数据库读取 jsonb 数据
func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	*j = JSONB(bytes)
	return nil
}

// Value 实现了 driver.Valuer 接口，允许将 jsonb 数据写入数据库
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// MarshalJSON 确保 JSONB 类型能被正确地序列化为 JSON
func (j JSONB) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON 确保 JSONB 类型能被正确地从 JSON 中反序列化
func (j *JSONB) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}
