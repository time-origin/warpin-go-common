package condition

import (
	"reflect"
	"regexp"
	"strings"
)

var orderByFieldPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// BaseCondition 提供了通用的分页和排序查询参数。
type BaseCondition struct {
	Page         *int   `json:"page,omitempty" form:"page"`
	Limit        *int   `json:"limit,omitempty" form:"limit"`
	OrderByField string `json:"order_by_field,omitempty" form:"order_by_field"`
	OrderBy      string `json:"order_by,omitempty" form:"order_by"` // "asc" or "desc"
}

// GetPage 返回页码，如果未提供则默认为 1。
func (c *BaseCondition) GetPage() int {
	if c.Page == nil || *c.Page <= 0 {
		return 1
	}
	return *c.Page
}

// GetLimit 返回每页数量，如果未提供则默认为 10。
func (c *BaseCondition) GetLimit() int {
	if c.Limit == nil || *c.Limit <= 0 {
		return 10
	}
	return *c.Limit
}

// GetOrderBy 返回排序字符串。
func (c *BaseCondition) GetOrderBy() string {
	field := strings.TrimSpace(c.OrderByField)
	if field == "" || !orderByFieldPattern.MatchString(field) {
		return ""
	}

	order := "asc"
	if strings.EqualFold(strings.TrimSpace(c.OrderBy), "desc") {
		order = "desc"
	}

	// 只允许单个 SQL 标识符，并对其加双引号，避免客户端排序参数成为 SQL 片段。
	return `"` + field + `" ` + order
}

// BuildQueryMap 从任何包含 BaseCondition 的结构体中构建一个只包含过滤条件的 map。
// 它会自动排除分页和排序字段。
func BuildQueryMap(conditionStruct interface{}) map[string]interface{} {
	queryMap := make(map[string]interface{})
	val := reflect.ValueOf(conditionStruct)
	typ := val.Type()

	// 如果是指针，获取其指向的元素
	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = val.Type()
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过 BaseCondition 结构体本身
		if fieldType.Type == reflect.TypeOf(BaseCondition{}) {
			continue
		}

		// 跳过零值字段 (例如，未提供的 *string 是 nil)
		if field.IsZero() {
			continue
		}

		// 获取 json tag 作为 map 的 key
		jsonTag := fieldType.Tag.Get("json")
		jsonFieldName := strings.Split(jsonTag, ",")[0]
		if jsonFieldName == "" || jsonFieldName == "-" {
			continue
		}

		// 对于指针类型，使用其指向的值
		if field.Kind() == reflect.Ptr {
			queryMap[jsonFieldName] = field.Elem().Interface()
		} else {
			queryMap[jsonFieldName] = field.Interface()
		}
	}

	return queryMap
}
