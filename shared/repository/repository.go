// Package repository provides generic repository implementations for database operations.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"oil/infras/otel"
	"oil/infras/postgres"
	"oil/shared/constant"
	"oil/shared/dto"
	"oil/shared/logger"
	"reflect"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx"
)

var (
	errRequiredFilter = errors.New("required filter")
)

type column struct {
	name  string
	table string
	alias string
}

type execer interface {
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
}

// Repository is a generic repository struct that provides common database operations for any entity type T.
type Repository[T any] struct {
	db            *postgres.Connection
	otel          otel.Otel
	table         string
	entity        string
	primaryColumn string
	columns       []column
	join          string
	InsertColumns []string
	selectQuery   string
	scopePrefix   string
}

// NewRepository creates a new instance of Repository for a specific entity type T,
// initializing it with the necessary database connection, table name, primary column, and OpenTelemetry instance for tracing.
// It uses reflection to determine the columns of the entity and any join queries defined by the entity's methods.
func NewRepository[T any](entityName, tableName, primaryColumn string, dbConnection *postgres.Connection, otl otel.Otel) Repository[T] {
	var zero T

	reflectType := reflect.TypeOf(zero)
	columns, insertColumns := getColumns(tableName, reflectType)

	valueOf := reflect.ValueOf(zero)
	method := valueOf.MethodByName("GetJoinQuery")
	joinQueryStr := ""

	if method.IsValid() {
		joinQuery := method.Call([]reflect.Value{})

		if len(joinQuery) > 0 {
			joinQueryStr = joinQuery[0].String()
		}
	}

	return Repository[T]{
		db:            dbConnection,
		otel:          otl,
		table:         tableName,
		entity:        entityName,
		primaryColumn: primaryColumn,
		columns:       columns,
		join:          joinQueryStr,
		InsertColumns: insertColumns,
		selectQuery:   buildSelectQuery(columns),
		scopePrefix:   constant.OtelRepositoryScopeName + "." + entityName + ".",
	}
}

// insert is a helper method that performs the actual insertion of a record into the database,
// abstracting the common logic for both Insert and InsertTx methods. It takes a context,
// an execer (which can be either a sqlx.DB or sqlx.Tx), and the model to be inserted, returning an error if the operation fails.
func (repo *Repository[T]) insert(ctx context.Context, exec execer, model T) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"insert")
	defer scope.End()

	placeholders := make([]string, 0, len(repo.InsertColumns))

	for _, col := range repo.InsertColumns {
		placeholders = append(placeholders, ":"+col)
	}

	query := "INSERT INTO " + repo.table + " (" + strings.Join(repo.InsertColumns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
	scope.SetAttribute(constant.OtelQueryAttributeKey, query)

	_, err := exec.NamedExecContext(ctx, query, model)
	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return fmt.Errorf("failed to insert data (%s): %w", repo.entity, err)
	}

	return nil
}

// Insert inserts a new record into the database, returning an error if the operation fails.
func (repo *Repository[T]) Insert(ctx context.Context, model T) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"Insert")
	defer scope.End()

	return repo.insert(ctx, repo.db.Write, model) //nolint:wrapcheck
}

// InsertTx inserts a new record into the database within the context of an existing transaction,
// allowing for atomic operations and ensuring that the insert operation is part of the transaction's scope.
// It takes a context, a sqlx.Tx transaction object, and the model to be inserted, returning an error if the operation fails.
func (repo *Repository[T]) InsertTx(ctx context.Context, sqltx *sqlx.Tx, model T) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"InsertTx")
	defer scope.End()

	return repo.insert(ctx, sqltx, model) //nolint:wrapcheck
}

// Exist checks if a record exists that matches the given filter criteria, returning true if it exists and false otherwise, along with any error encountered during the operation.
func (repo *Repository[T]) Exist(ctx context.Context, filter dto.FilterGroup) (bool, error) {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"Exist")
	defer scope.End()

	where, args := repo.BuildWhereClause(ctx, filter)
	if where == "" {
		return false, errRequiredFilter
	}

	query := "SELECT EXISTS(SELECT 1 FROM " + repo.table + " " + where + ")"
	scope.SetAttribute(constant.OtelQueryAttributeKey, query)

	exist := false

	namedQuery, namedArgs, err := sqlx.Named(query, args)
	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return false, fmt.Errorf("failed to check exist data (%s): %w", repo.entity, err)
	}

	namedQuery = repo.db.Read.Rebind(namedQuery)

	err = repo.db.Read.GetContext(ctx, &exist, namedQuery, namedArgs...)
	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return false, fmt.Errorf("failed to check exist data (%s): %w", repo.entity, err)
	}

	return exist, nil
}

// Get returns a single record that matches the given filter criteria, along with any error encountered during the operation.
func (repo *Repository[T]) Get(ctx context.Context, filter dto.FilterGroup, columns ...string) (T, error) {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"Get")
	defer scope.End()

	where, args := repo.BuildWhereClause(ctx, filter)
	selectQuery := repo.getSelectQuery(ctx, columns...)

	query := "SELECT " + selectQuery + " FROM " + repo.table + " " + repo.join + " " + where
	scope.SetAttribute(constant.OtelQueryAttributeKey, query)

	var model T

	namedQuery, namedArgs, err := sqlx.Named(query, args)
	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return model, fmt.Errorf("failed to prepare statement (%s): %w", repo.entity, err)
	}

	namedQuery = repo.db.Read.Rebind(namedQuery)

	err = repo.db.Read.GetContext(ctx, &model, namedQuery, namedArgs...)
	if errors.Is(err, sql.ErrNoRows) {
		return model, nil
	}

	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return model, fmt.Errorf("failed to get data (%s): %w", repo.entity, err)
	}

	return model, nil
}

// GetAll returns a slice of records that match the given filter criteria, along with any error encountered during the operation.
func (repo *Repository[T]) GetAll(ctx context.Context, params dto.QueryParams, filter dto.FilterGroup, columns ...string) ([]T, error) {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"GetAll")
	defer scope.End()

	where, args := repo.BuildWhereClause(ctx, filter)
	selectQuery := repo.getSelectQuery(ctx, columns...)

	var ordering, pagination string

	page := params.Page
	limit := params.Limit

	if page > 0 && limit > 0 {
		args["limit"] = limit
		args["offset"] = (page - 1) * limit

		pagination = "LIMIT :limit OFFSET :offset"
	} else if limit > 0 {
		args["limit"] = limit

		pagination = "LIMIT :limit"
	}

	ordering = repo.buildOrderBy(params)

	query := "SELECT " + selectQuery + " FROM " + repo.table + " " + repo.join + " " + where + " " + ordering + " " + pagination

	scope.SetAttribute(constant.OtelQueryAttributeKey, query)

	var models []T

	namedQuery, namedArgs, err := sqlx.Named(query, args)
	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return models, fmt.Errorf("failed to prepare statement (%s): %w", repo.entity, err)
	}

	namedQuery = repo.db.Read.Rebind(namedQuery)

	err = repo.db.Read.SelectContext(ctx, &models, namedQuery, namedArgs...)
	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return models, fmt.Errorf("failed to get all data (%s): %w", repo.entity, err)
	}

	return models, nil
}

// Count returns the total number of records that match the given filter criteria.
func (repo *Repository[T]) Count(ctx context.Context, filter dto.FilterGroup) (int, error) {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"Count")
	defer scope.End()

	where, args := repo.BuildWhereClause(ctx, filter)

	query := "SELECT COUNT(" + repo.table + "." + repo.primaryColumn + ") FROM " + repo.table + " " + repo.join + " " + where
	scope.SetAttribute(constant.OtelQueryAttributeKey, query)

	var count int

	namedQuery, namedArgs, err := sqlx.Named(query, args)
	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return 0, fmt.Errorf("failed to prepare statement (%s): %w", repo.entity, err)
	}

	namedQuery = repo.db.Read.Rebind(namedQuery)

	err = repo.db.Read.GetContext(ctx, &count, namedQuery, namedArgs...)
	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return 0, fmt.Errorf("failed to count data (%s): %w", repo.entity, err)
	}

	return count, nil
}

func (repo *Repository[T]) delete(ctx context.Context, exec execer, filter dto.FilterGroup) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"delete")
	defer scope.End()

	where, args := repo.BuildWhereClause(ctx, filter)
	if where == "" {
		return errRequiredFilter
	}

	query := "DELETE FROM " + repo.table + " " + where
	scope.SetAttribute(constant.OtelQueryAttributeKey, query)

	_, err := exec.NamedExecContext(ctx, query, args)
	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return fmt.Errorf("failed to delete data (%s): %w", repo.entity, err)
	}

	return nil
}

// Delete removes records matching the filter from the database.
func (repo *Repository[T]) Delete(ctx context.Context, filter dto.FilterGroup) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"Delete")
	defer scope.End()

	return repo.delete(ctx, repo.db.Write, filter) //nolint:wrapcheck
}

// DeleteTx removes records matching the filter within a transaction.
func (repo *Repository[T]) DeleteTx(ctx context.Context, sqltx *sqlx.Tx, filter dto.FilterGroup) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"DeleteTx")
	defer scope.End()

	return repo.delete(ctx, sqltx, filter) //nolint:wrapcheck
}

func (repo *Repository[T]) update(ctx context.Context, exec execer, mod map[string]any, filter dto.FilterGroup) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"update")
	defer scope.End()

	updateQuery, setArgs := buildUpdateAssignments(mod)

	where, args := repo.BuildWhereClause(ctx, filter)
	query := "UPDATE " + repo.table + " SET " + updateQuery + " " + where

	scope.SetAttribute(constant.OtelQueryAttributeKey, query)
	maps.Copy(args, setArgs)

	_, err := exec.NamedExecContext(ctx, query, args)
	if err != nil {
		logger.ErrorWithStack(err)
		scope.TraceError(err)

		return fmt.Errorf("failed to update data (%s): %w", repo.entity, err)
	}

	return nil
}

// Update modifies records matching the filter with the provided values.
func (repo *Repository[T]) Update(ctx context.Context, mod map[string]any, filter dto.FilterGroup) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"Update")
	defer scope.End()

	return repo.update(ctx, repo.db.Write, mod, filter) //nolint:wrapcheck
}

// UpdateTx modifies records matching the filter within a transaction.
func (repo *Repository[T]) UpdateTx(ctx context.Context, sqltx *sqlx.Tx, mod map[string]any, filter dto.FilterGroup) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"UpdateTx")
	defer scope.End()

	return repo.update(ctx, sqltx, mod, filter) //nolint:wrapcheck
}

// InsertBulk inserts multiple records into the database.
func (repo *Repository[T]) InsertBulk(ctx context.Context, models []T) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"InsertBulk")
	defer scope.End()

	return repo.insertBulk(ctx, repo.db.Write, models)
}

// InsertBulkTx inserts multiple records into the database within a transaction.
func (repo *Repository[T]) InsertBulkTx(ctx context.Context, sqltx *sqlx.Tx, models []T) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"InsertBulkTx")
	defer scope.End()

	return repo.insertBulk(ctx, sqltx, models)
}

func (repo *Repository[T]) insertBulk(ctx context.Context, exec execer, models []T) error {
	ctx, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"insertBulk")
	defer scope.End()

	var err error

	placeholders := make([]string, 0, len(models))

	for _, column := range repo.InsertColumns {
		placeholders = append(placeholders, ":"+column)
	}

	query := "INSERT INTO " + repo.table + " (" + strings.Join(repo.InsertColumns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"

	scope.SetAttribute(constant.OtelQueryAttributeKey+repo.entity, query)

	_, err = exec.NamedExecContext(ctx, query, models)
	if err != nil {
		scope.TraceError(err)
		logger.ErrorWithStack(err)

		return fmt.Errorf("failed to bulk insert order: %w", err)
	}

	return nil
}

// buildUpdateAssignments builds the SET clause for an UPDATE statement and the
// named args backing it. SET placeholders are prefixed with "_set_" so they can
// never collide with WHERE-clause args that reference the same column name.
func buildUpdateAssignments(mod map[string]any) (clause string, args map[string]any) {
	args = make(map[string]any, len(mod))

	var b strings.Builder

	const estimateLen = 30 // ponytail: "col = :_set_col, " per entry
	b.Grow(len(mod) * estimateLen)

	first := true

	for col, val := range mod {
		setParam := "_set_" + col

		if !first {
			b.WriteString(", ")
		}

		b.WriteString(col)
		b.WriteString(" = :")
		b.WriteString(setParam)
		args[setParam] = val
		first = false
	}

	return b.String(), args
}

// buildOrderBy returns a safe ORDER BY clause, or an empty string when no valid
// ordering is requested. SortBy is validated against the entity's known columns
// to prevent SQL injection through the raw sort field; SortDir is already
// constrained to ASC/DESC upstream.
func (repo *Repository[T]) buildOrderBy(params dto.QueryParams) string {
	if params.SortBy == "" || params.SortDir == "" {
		return ""
	}

	sortable := slices.ContainsFunc(repo.columns, func(c column) bool {
		return c.name == params.SortBy || c.alias == params.SortBy
	})
	if !sortable {
		return ""
	}

	return "ORDER BY " + params.SortBy + " " + params.SortDir
}

// buildSelectQuery pre-computes the full SELECT column string from a column list.
func buildSelectQuery(columns []column) string {
	parts := make([]string, len(columns))
	for i, c := range columns {
		switch {
		case c.table == "":
			parts[i] = c.name
		case c.alias != "":
			parts[i] = c.table + "." + c.name + " AS " + c.alias
		default:
			parts[i] = c.table + "." + c.name
		}
	}

	return strings.Join(parts, ", ")
}

func (repo *Repository[T]) getSelectQuery(ctx context.Context, columnsParam ...string) string {
	_, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"getSelectQuery")
	defer scope.End()

	if len(columnsParam) == 0 {
		return repo.selectQuery
	}

	columns := make([]string, 0, len(repo.columns))
	for _, col := range repo.columns {
		if !slices.Contains(columnsParam, col.name) {
			continue
		}

		var column string

		switch {
		case col.table == "":
			column = col.name
		case col.alias != "":
			column = col.table + "." + col.name + " AS " + col.alias
		default:
			column = col.table + "." + col.name
		}

		columns = append(columns, column)
	}

	return strings.Join(columns, ", ")
}

// BuildWhereClause builds a WHERE clause from the provided filter.
func (repo *Repository[T]) BuildWhereClause(ctx context.Context, filter dto.FilterGroup) (string, map[string]any) {
	_, scope := repo.otel.NewScope(ctx, constant.OtelRepositoryScopeName, repo.scopePrefix+"BuildWhereClause")
	defer scope.End()

	where, args := filter.GetWhereClause()

	if where == "" {
		return where, map[string]any{}
	}

	return " WHERE " + where + " ", args
}

func getColumns(table string, reflectType reflect.Type) (columns []column, insertColumns []string) {
	for i := range reflectType.NumField() {
		field := reflectType.Field(i)
		dbTag := field.Tag.Get("db")
		tableField := field.Tag.Get("table")
		colTag := field.Tag.Get("column")

		if tableField == "" {
			tableField = table
		}

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			col, insertCol := getColumns(table, field.Type)
			columns = append(columns, col...)
			insertColumns = append(insertColumns, insertCol...)
		}

		if dbTag == "" {
			continue
		}

		if tableField == table {
			insertColumns = append(insertColumns, dbTag)
		}

		if colTag == "" {
			columns = append(columns, column{name: dbTag, table: tableField})
		} else {
			columns = append(columns, column{name: colTag, table: tableField, alias: dbTag})
		}
	}

	return columns, insertColumns
}
