package repository

import (
	"context"
	"reflect"
	"testing"
	"time"

	otelMocks "oil/infras/otel/mocks"
	postgresMocks "oil/infras/postgres/mocks"
	"oil/shared/dto"
	sharedModel "oil/shared/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEntityName = "test"
	testTableName  = "test_table"
	testPrimaryCol = "id"
)

type testEntity struct {
	ID   string `db:"id"`
	Name string `db:"name"`
	sharedModel.Metadata
}

// GetJoinQuery satisfies the optional join hook used by NewRepository.
func (testEntity) GetJoinQuery() string { return "" }

func newTestRepo(t *testing.T) (Repository[testEntity], sqlmock.Sqlmock) {
	t.Helper()

	_, mock, conn := postgresMocks.SetupPostgresConnection(t)
	repo := NewRepository[testEntity](testEntityName, testTableName, testPrimaryCol, conn, otelMocks.NewOtel())

	return repo, mock
}

func idFilter(id string) dto.FilterGroup {
	return dto.FilterGroup{
		Operator: dto.FilterGroupOperatorAnd,
		Filters: []any{
			dto.Filter{Field: "id", Operator: dto.FilterOperatorEq, Value: id, Table: testTableName},
		},
	}
}

func testRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "name", "created_at", "modified_at", "created_by", "modified_by"})
}

// --- M3: reads must not prepare-then-discard a statement ---

func TestGetReadsWithoutPrepare(t *testing.T) {
	repo, mock := newTestRepo(t)

	rows := testRows().AddRow("1", "alice", time.Time{}, time.Time{}, "", "")
	mock.ExpectQuery("SELECT .* FROM test_table").WillReturnRows(rows)

	got, err := repo.Get(context.Background(), idFilter("1"))
	require.NoError(t, err)
	assert.Equal(t, "1", got.ID)
	assert.Equal(t, "alice", got.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllReadsWithoutPrepare(t *testing.T) {
	repo, mock := newTestRepo(t)

	rows := testRows().
		AddRow("1", "alice", time.Time{}, time.Time{}, "", "").
		AddRow("2", "bob", time.Time{}, time.Time{}, "", "")
	mock.ExpectQuery("SELECT .* FROM test_table").WillReturnRows(rows)

	got, err := repo.GetAll(context.Background(), dto.QueryParams{Page: 1, Limit: 10}, dto.FilterGroup{})
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountReadsWithoutPrepare(t *testing.T) {
	repo, mock := newTestRepo(t)

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	got, err := repo.Count(context.Background(), dto.FilterGroup{})
	require.NoError(t, err)
	assert.Equal(t, 5, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExistReadsWithoutPrepare(t *testing.T) {
	repo, mock := newTestRepo(t)

	mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	got, err := repo.Exist(context.Background(), idFilter("1"))
	require.NoError(t, err)
	assert.True(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- M2 regression anchor: SortBy injection ---

func TestBuildOrderBy(t *testing.T) {
	repo, _ := newTestRepo(t)

	tests := []struct {
		name   string
		params dto.QueryParams
		want   string
	}{
		{"known column asc", dto.QueryParams{SortBy: "name", SortDir: dto.SortDirAsc}, "ORDER BY name ASC"},
		{"known column desc", dto.QueryParams{SortBy: "id", SortDir: dto.SortDirDesc}, "ORDER BY id DESC"},
		{"embedded metadata column", dto.QueryParams{SortBy: "created_at", SortDir: dto.SortDirDesc}, "ORDER BY created_at DESC"},
		{"empty params", dto.QueryParams{}, ""},
		{"missing direction", dto.QueryParams{SortBy: "name"}, ""},
		{"unknown column", dto.QueryParams{SortBy: "nope", SortDir: dto.SortDirAsc}, ""},
		{"sql injection attempt", dto.QueryParams{SortBy: "id; DROP TABLE users", SortDir: dto.SortDirAsc}, ""},
		{"injection via subquery", dto.QueryParams{SortBy: "(SELECT pg_sleep(10))", SortDir: dto.SortDirAsc}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.buildOrderBy(tt.params)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- M1 regression anchor: update param collision ---

func TestBuildUpdateAssignments(t *testing.T) {
	t.Run("prefixes set params so they do not collide with filter args", func(t *testing.T) {
		clause, args := buildUpdateAssignments(map[string]any{"status": "active"})

		assert.Equal(t, "status = :_set_status", clause)
		assert.Equal(t, "active", args["_set_status"])

		_, collides := args["status"]
		assert.False(t, collides, "set arg must not use the bare column name")
	})

	t.Run("keeps set and filter values distinct when column names match", func(t *testing.T) {
		// Filter says status = "pending"; update sets status = "active".
		filterArgs := map[string]any{"status": "pending"}

		clause, setArgs := buildUpdateAssignments(map[string]any{"status": "active"})
		for k, v := range setArgs {
			filterArgs[k] = v
		}

		assert.Contains(t, clause, "_set_status")
		assert.Equal(t, "pending", filterArgs["status"], "filter value must survive")
		assert.Equal(t, "active", filterArgs["_set_status"], "set value must survive")
	})
}

// --- Safety-net characterization tests (existing behavior) ---

func TestGetColumns(t *testing.T) {
	columns, insertColumns := getColumns(testTableName, reflect.TypeOf(testEntity{}))

	names := make([]string, 0, len(columns))
	for _, c := range columns {
		names = append(names, c.name)
	}

	assert.Contains(t, names, "id")
	assert.Contains(t, names, "name")
	assert.Contains(t, names, "created_at")
	assert.Contains(t, insertColumns, "id")
	assert.Contains(t, insertColumns, "name")
}

func TestGetSelectQuery(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	all := repo.getSelectQuery(ctx)
	assert.Contains(t, all, testTableName+".id")
	assert.Contains(t, all, testTableName+".name")

	subset := repo.getSelectQuery(ctx, "id")
	assert.Contains(t, subset, testTableName+".id")
	assert.NotContains(t, subset, testTableName+".name")
}

func TestBuildWhereClause(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	emptyWhere, emptyArgs := repo.BuildWhereClause(ctx, dto.FilterGroup{})
	assert.Equal(t, "", emptyWhere)
	assert.Empty(t, emptyArgs)

	filter := dto.FilterGroup{
		Operator: dto.FilterGroupOperatorAnd,
		Filters: []any{
			dto.Filter{Field: "id", Operator: dto.FilterOperatorEq, Value: "abc", Table: testTableName},
		},
	}
	where, args := repo.BuildWhereClause(ctx, filter)
	assert.Contains(t, where, "WHERE")
	assert.Contains(t, where, "test_table.id = :id")
	assert.Equal(t, "abc", args["id"])
}

func TestExistRequiresFilter(t *testing.T) {
	repo, _ := newTestRepo(t)

	_, err := repo.Exist(context.Background(), dto.FilterGroup{})
	assert.ErrorIs(t, err, errRequiredFilter)
}

func TestDeleteRequiresFilter(t *testing.T) {
	repo, _ := newTestRepo(t)

	err := repo.Delete(context.Background(), dto.FilterGroup{})
	assert.ErrorIs(t, err, errRequiredFilter)
}

func TestInsertHappyPath(t *testing.T) {
	repo, mock := newTestRepo(t)

	mock.ExpectExec("INSERT INTO test_table").WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Insert(context.Background(), testEntity{ID: "1", Name: "x"})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

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
