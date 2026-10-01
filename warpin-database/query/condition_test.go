package condition

import "testing"

func TestBaseConditionGetOrderBy(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		direction string
		want      string
	}{
		{
			name:      "valid ascending field",
			field:     "created_at",
			direction: "asc",
			want:      `"created_at" asc`,
		},
		{
			name:      "valid descending field ignores case and spaces",
			field:     "position",
			direction: " DESC ",
			want:      `"position" desc`,
		},
		{
			name:      "unknown direction remains ascending",
			field:     "id",
			direction: "sideways",
			want:      `"id" asc`,
		},
		{
			name:      "blank field has no order",
			field:     "  ",
			direction: "desc",
			want:      "",
		},
		{
			name:      "sql suffix is rejected",
			field:     "id desc; drop table users",
			direction: "asc",
			want:      "",
		},
		{
			name:      "multiple columns are rejected",
			field:     "position,id",
			direction: "desc",
			want:      "",
		},
		{
			name:      "quoted field is rejected",
			field:     `position"`,
			direction: "desc",
			want:      "",
		},
		{
			name:      "qualified expression is rejected",
			field:     "users.id",
			direction: "desc",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := &BaseCondition{
				OrderByField: tt.field,
				OrderBy:      tt.direction,
			}

			if got := condition.GetOrderBy(); got != tt.want {
				t.Fatalf("GetOrderBy() = %q, want %q", got, tt.want)
			}
		})
	}
}
