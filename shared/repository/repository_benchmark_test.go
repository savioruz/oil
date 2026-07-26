package repository

import (
	"context"
	"testing"

	otelMocks "oil/infras/otel/mocks"
	"oil/shared/dto"

	"github.com/stretchr/testify/require"
)

func BenchmarkGetSelectQuery_zeroParams(b *testing.B) {
	repo := makeTestRepo()
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		_ = repo.getSelectQuery(ctx)
	}
}

func BenchmarkGetSelectQuery_withParams(b *testing.B) {
	repo := makeTestRepo()
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		_ = repo.getSelectQuery(ctx, "id", "name")
	}
}

func BenchmarkBuildWhereClause_singleEq(b *testing.B) {
	repo := makeTestRepo()
	fg := dto.FilterGroup{
		Operator: dto.FilterGroupOperatorAnd,
		Filters: []any{
			dto.Filter{Field: "id", Operator: dto.FilterOperatorEq, Value: "abc", Table: testTableName},
		},
	}
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		repo.BuildWhereClause(ctx, fg)
	}
}

func BenchmarkBuildWhereClause_singleIn(b *testing.B) {
	repo := makeTestRepo()
	fg := dto.FilterGroup{
		Operator: dto.FilterGroupOperatorAnd,
		Filters: []any{
			dto.Filter{Field: "id", Operator: dto.FilterOperatorIn, Value: []string{"a", "b", "c", "d", "e"}, Table: testTableName},
		},
	}
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		repo.BuildWhereClause(ctx, fg)
	}
}

func BenchmarkBuildUpdateAssignments_3cols(b *testing.B) {
	mod := map[string]any{"name": "alice", "email": "a@b.com", "status": "active"}
	b.ResetTimer()
	for range b.N {
		_, _ = buildUpdateAssignments(mod)
	}
}

func BenchmarkBuildUpdateAssignments_1col(b *testing.B) {
	mod := map[string]any{"name": "alice"}
	b.ResetTimer()
	for range b.N {
		_, _ = buildUpdateAssignments(mod)
	}
}

// makeTestRepo creates a lightweight Repository for micro-benchmarks.
// It bypasses the full NewRepository<T> because we only need its helper methods.
func makeTestRepo() *Repository[testEntity] {
	return &Repository[testEntity]{
		otel:          otelMocks.NewOtel(),
		table:         testTableName,
		entity:        testEntityName,
		primaryColumn: testPrimaryCol,
		selectQuery:   "test_table.id AS id, test_table.name AS name",
		columns: []column{
			{table: testTableName, name: "id", alias: "id"},
			{table: testTableName, name: "name", alias: "name"},
		},
	}
}

// prevent compiler optimizations
var (
	_ string
	_ map[string]any
)

func init() {
	_, _ = require.New(nil), context.Background()
}
