package dto_test

import (
	"github.com/savioruz/oil/shared/constant"
	"github.com/savioruz/oil/shared/dto"
	"github.com/savioruz/oil/shared/model"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestMetadata_FromModel(t *testing.T) {
	// Create test time values
	createdAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	modifiedAt := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)

	modelMetadata := model.Metadata{
		CreatedAt:  createdAt,
		ModifiedAt: modifiedAt,
		CreatedBy:  "creator",
		ModifiedBy: "modifier",
	}

	metadata := &dto.Metadata{}
	metadata.FromModel(modelMetadata)

	expectedCreatedAt := createdAt.Format(constant.DateFormat)
	expectedModifiedAt := modifiedAt.Format(constant.DateFormat)

	if metadata.CreatedAt != expectedCreatedAt {
		t.Errorf("expected CreatedAt to be %s, got %s", expectedCreatedAt, metadata.CreatedAt)
	}

	if metadata.ModifiedAt != expectedModifiedAt {
		t.Errorf("expected ModifiedAt to be %s, got %s", expectedModifiedAt, metadata.ModifiedAt)
	}

	if metadata.CreatedBy != "creator" {
		t.Errorf("expected CreatedBy to be 'creator', got %s", metadata.CreatedBy)
	}

	if metadata.ModifiedBy != "modifier" {
		t.Errorf("expected ModifiedBy to be 'modifier', got %s", metadata.ModifiedBy)
	}
}

func TestQueryParams_FromRequest(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		defaultRequest bool
		expected       dto.QueryParams
	}{
		{
			name: "with all valid parameters",
			queryParams: map[string]string{
				"page":     "2",
				"limit":    "20",
				"sort_by":  "name",
				"sort_dir": "ASC",
			},
			defaultRequest: false,
			expected: dto.QueryParams{
				Page:    2,
				Limit:   20,
				SortBy:  "name",
				SortDir: "ASC",
			},
		},
		{
			name:           "with default request enabled and no parameters",
			queryParams:    map[string]string{},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage,
				Limit:   constant.DefaultValueLimit,
				SortBy:  "",
				SortDir: "",
			},
		},
		{
			name:           "with default request disabled and no parameters",
			queryParams:    map[string]string{},
			defaultRequest: false,
			expected: dto.QueryParams{
				Page:    0,
				Limit:   0,
				SortBy:  "",
				SortDir: "",
			},
		},
		{
			name: "with invalid page parameter",
			queryParams: map[string]string{
				"page": "invalid",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage, // Should use default
				Limit:   constant.DefaultValueLimit,
				SortBy:  "",
				SortDir: "",
			},
		},
		{
			name: "with negative page parameter",
			queryParams: map[string]string{
				"page": "-1",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage, // Should use default
				Limit:   constant.DefaultValueLimit,
				SortBy:  "",
				SortDir: "",
			},
		},
		{
			name: "with zero page parameter",
			queryParams: map[string]string{
				"page": "0",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage, // Should use default
				Limit:   constant.DefaultValueLimit,
				SortBy:  "",
				SortDir: "",
			},
		},
		{
			name: "with invalid limit parameter",
			queryParams: map[string]string{
				"limit": "invalid",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage,
				Limit:   constant.DefaultValueLimit, // Should use default
				SortBy:  "",
				SortDir: "",
			},
		},
		{
			name: "with negative limit parameter",
			queryParams: map[string]string{
				"limit": "-10",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage,
				Limit:   constant.DefaultValueLimit, // Should use default
				SortBy:  "",
				SortDir: "",
			},
		},
		{
			name: "with partial parameters and defaults enabled",
			queryParams: map[string]string{
				"page":    "3",
				"sort_by": "email",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    3,
				Limit:   constant.DefaultValueLimit, // Should use default
				SortBy:  "email",
				SortDir: "", // EmptyString when not provided
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a URL with query parameters
			baseURL := "http://example.com/test"
			u, err := url.Parse(baseURL)
			if err != nil {
				t.Fatalf("failed to parse URL: %v", err)
			}

			// Add query parameters
			query := u.Query()
			for key, value := range tt.queryParams {
				query.Set(key, value)
			}
			u.RawQuery = query.Encode()

			// Create HTTP request
			req, err := http.NewRequest("GET", u.String(), nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			// Test the method
			queryParams := &dto.QueryParams{}
			queryParams.FromRequest(req, tt.defaultRequest)

			// Verify results
			if queryParams.Page != tt.expected.Page {
				t.Errorf("expected Page to be %d, got %d", tt.expected.Page, queryParams.Page)
			}
			if queryParams.Limit != tt.expected.Limit {
				t.Errorf("expected Limit to be %d, got %d", tt.expected.Limit, queryParams.Limit)
			}
			if queryParams.SortBy != tt.expected.SortBy {
				t.Errorf("expected SortBy to be %s, got %s", tt.expected.SortBy, queryParams.SortBy)
			}
			if queryParams.SortDir != tt.expected.SortDir {
				t.Errorf("expected SortDir to be %s, got %s", tt.expected.SortDir, queryParams.SortDir)
			}
		})
	}
}

func TestSortDirectionConstants(t *testing.T) {
	if dto.SortDirAsc != "ASC" {
		t.Errorf("expected SortDirAsc to be 'ASC', got %s", dto.SortDirAsc)
	}
	if dto.SortDirDesc != "DESC" {
		t.Errorf("expected SortDirDesc to be 'DESC', got %s", dto.SortDirDesc)
	}
}

func BenchmarkFilterGetWhereClause_eq(b *testing.B) {
	f := dto.Filter{Field: "id", Operator: dto.FilterOperatorEq, Value: "abc", Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}

func BenchmarkFilterGetWhereClause_like(b *testing.B) {
	f := dto.Filter{Field: "name", Operator: dto.FilterOperatorLike, Value: "alice", Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}

func BenchmarkFilterGetWhereClause_in_5(b *testing.B) {
	f := dto.Filter{Field: "id", Operator: dto.FilterOperatorIn, Value: []string{"a", "b", "c", "d", "e"}, Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}

func BenchmarkFilterGetWhereClause_in_20(b *testing.B) {
	vals := make([]string, 20)
	for i := range 20 {
		vals[i] = "val_xxxxxxxxxxxxxxxxxx"
	}
	f := dto.Filter{Field: "id", Operator: dto.FilterOperatorIn, Value: vals, Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}

func BenchmarkFilterGetWhereClause_notEq(b *testing.B) {
	f := dto.Filter{Field: "status", Operator: dto.FilterOperatorNotEq, Value: "deleted", Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}

func BenchmarkFilterGroupGetWhereClause_3filters(b *testing.B) {
	fg := dto.FilterGroup{
		Operator: dto.FilterGroupOperatorAnd,
		Filters: []any{
			dto.Filter{Field: "id", Operator: dto.FilterOperatorEq, Value: "123", Table: "users"},
			dto.Filter{Field: "status", Operator: dto.FilterOperatorNotEq, Value: "deleted", Table: "users"},
			dto.Filter{Field: "name", Operator: dto.FilterOperatorLike, Value: "alice", Table: "users"},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		fg.GetWhereClause()
	}
}

func BenchmarkFilterIsNull(b *testing.B) {
	f := dto.Filter{Field: "deleted_at", Operator: dto.FilterIsNull, Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}

// --- Filter.GetWhereClause unit tests ---

func TestFilter_GetWhereClause(t *testing.T) {
	tests := []struct {
		name   string
		filter dto.Filter
		want   string
		args   map[string]any
	}{
		{
			name:   "eq with table prefix",
			filter: dto.Filter{Field: "id", Operator: dto.FilterOperatorEq, Value: "abc", Table: "users"},
			want:   "users.id = :id",
			args:   map[string]any{"id": "abc"},
		},
		{
			name:   "eq without table",
			filter: dto.Filter{Field: "status", Operator: dto.FilterOperatorEq, Value: "active"},
			want:   "status = :status",
			args:   map[string]any{"status": "active"},
		},
		{
			name:   "eq with custom ArgName",
			filter: dto.Filter{ArgName: "uid", Field: "id", Operator: dto.FilterOperatorEq, Value: "xyz"},
			want:   "id = :uid",
			args:   map[string]any{"uid": "xyz"},
		},
		{
			name:   "like with string",
			filter: dto.Filter{Field: "name", Operator: dto.FilterOperatorLike, Value: "alice", Table: "users"},
			want:   "LOWER(users.name) LIKE LOWER(:name) ",
			args:   map[string]any{"name": "%alice%"},
		},
		{
			name:   "like with non-string returns empty",
			filter: dto.Filter{Field: "name", Operator: dto.FilterOperatorLike, Value: 123},
			want:   "",
			args:   map[string]any{},
		},
		{
			name:   "in with []string",
			filter: dto.Filter{Field: "id", Operator: dto.FilterOperatorIn, Value: []string{"a", "b"}},
			want:   "id IN (:id_0, :id_1) ",
			args:   map[string]any{"id_0": "a", "id_1": "b"},
		},
		{
			name:   "in with []int",
			filter: dto.Filter{Field: "id", Operator: dto.FilterOperatorIn, Value: []int{1, 2}},
			want:   "id IN (:id_0, :id_1) ",
			args:   map[string]any{"id_0": 1, "id_1": 2},
		},
		{
			name:   "in with []int64",
			filter: dto.Filter{Field: "id", Operator: dto.FilterOperatorIn, Value: []int64{10, 20}},
			want:   "id IN (:id_0, :id_1) ",
			args:   map[string]any{"id_0": int64(10), "id_1": int64(20)},
		},
		{
			name:   "in with []any",
			filter: dto.Filter{Field: "id", Operator: dto.FilterOperatorIn, Value: []any{"x", 42}},
			want:   "id IN (:id_0, :id_1) ",
			args:   map[string]any{"id_0": "x", "id_1": 42},
		},
		{
			name:   "not_eq",
			filter: dto.Filter{Field: "status", Operator: dto.FilterOperatorNotEq, Value: "deleted", Table: "t"},
			want:   "t.status != :status",
			args:   map[string]any{"status": "deleted"},
		},
		{
			name:   "less_eq",
			filter: dto.Filter{Field: "age", Operator: dto.FilterOperatorLessEq, Value: 65},
			want:   "age <= :age",
			args:   map[string]any{"age": 65},
		},
		{
			name:   "greater_eq",
			filter: dto.Filter{Field: "score", Operator: dto.FilterOperatorGreaterEq, Value: 10},
			want:   "score >= :score",
			args:   map[string]any{"score": 10},
		},
		{
			name:   "plan (plain query)",
			filter: dto.Filter{Field: "", Operator: dto.FilterPlainQuery, Value: "a = :a AND b = :b"},
			want:   "(a = :a AND b = :b)",
			args:   map[string]any{},
		},
		{
			name:   "is_not_null",
			filter: dto.Filter{Field: "deleted_at", Operator: dto.FilterIsNotNull, Table: "users"},
			want:   "users.deleted_at IS NOT NULL",
			args:   map[string]any{},
		},
		{
			name:   "is_null",
			filter: dto.Filter{Field: "deleted_at", Operator: dto.FilterIsNull},
			want:   "deleted_at IS NULL",
			args:   map[string]any{},
		},
		{
			name:   "unknown operator returns empty",
			filter: dto.Filter{Field: "x", Operator: "bogus", Value: 1},
			want:   "",
			args:   map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWhere, gotArgs := tt.filter.GetWhereClause()

			if gotWhere != tt.want {
				t.Errorf("where = %q, want %q", gotWhere, tt.want)
			}
			if len(gotArgs) != len(tt.args) {
				t.Fatalf("arg count = %d, want %d (got %v)", len(gotArgs), len(tt.args), gotArgs)
			}
			for k, v := range tt.args {
				gv, ok := gotArgs[k]
				if !ok {
					t.Errorf("missing arg %q", k)
				}
				if gv != v {
					t.Errorf("arg[%q] = %v, want %v", k, gv, v)
				}
			}
		})
	}
}

// --- FilterGroup.GetWhereClause unit tests ---

func TestFilterGroup_GetWhereClause(t *testing.T) {
	tests := []struct {
		name string
		fg   dto.FilterGroup
		want string
		args map[string]any
	}{
		{
			name: "empty",
			fg:   dto.FilterGroup{Operator: dto.FilterGroupOperatorAnd},
			want: "",
			args: map[string]any{},
		},
		{
			name: "single filter",
			fg: dto.FilterGroup{
				Operator: dto.FilterGroupOperatorAnd,
				Filters:  []any{dto.Filter{Field: "id", Operator: dto.FilterOperatorEq, Value: "1", Table: "t"}},
			},
			want: "(t.id = :id)",
			args: map[string]any{"id": "1"},
		},
		{
			name: "two filters AND",
			fg: dto.FilterGroup{
				Operator: dto.FilterGroupOperatorAnd,
				Filters: []any{
					dto.Filter{Field: "a", Operator: dto.FilterOperatorEq, Value: "1"},
					dto.Filter{Field: "b", Operator: dto.FilterOperatorEq, Value: "2"},
				},
			},
			want: "(a = :a AND b = :b)",
			args: map[string]any{"a": "1", "b": "2"},
		},
		{
			name: "two filters OR",
			fg: dto.FilterGroup{
				Operator: dto.FilterGroupOperatorOr,
				Filters: []any{
					dto.Filter{Field: "x", Operator: dto.FilterOperatorEq, Value: "1"},
					dto.Filter{Field: "y", Operator: dto.FilterOperatorEq, Value: "2"},
				},
			},
			want: "(x = :x OR y = :y)",
			args: map[string]any{"x": "1", "y": "2"},
		},
		{
			name: "nested FilterGroup",
			fg: dto.FilterGroup{
				Operator: dto.FilterGroupOperatorAnd,
				Filters: []any{
					dto.Filter{Field: "a", Operator: dto.FilterOperatorEq, Value: "1"},
					dto.FilterGroup{
						Operator: dto.FilterGroupOperatorOr,
						Filters: []any{
							dto.Filter{Field: "b", Operator: dto.FilterOperatorEq, Value: "2"},
							dto.Filter{Field: "c", Operator: dto.FilterOperatorEq, Value: "3"},
						},
					},
				},
			},
			want: "(a = :a AND (b = :b OR c = :c))",
			args: map[string]any{"a": "1", "b": "2", "c": "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWhere, gotArgs := tt.fg.GetWhereClause()

			if gotWhere != tt.want {
				t.Errorf("where = %q, want %q", gotWhere, tt.want)
			}
			if len(gotArgs) != len(tt.args) {
				t.Fatalf("arg count = %d, want %d (got %v)", len(gotArgs), len(tt.args), gotArgs)
			}
			for k, v := range tt.args {
				if gotArgs[k] != v {
					t.Errorf("arg[%q] = %v, want %v", k, gotArgs[k], v)
				}
			}
		})
	}
}
