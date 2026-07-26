package dto

import "testing"

func BenchmarkFilterGetWhereClause_eq(b *testing.B) {
	f := Filter{Field: "id", Operator: FilterOperatorEq, Value: "abc", Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}

func BenchmarkFilterGetWhereClause_like(b *testing.B) {
	f := Filter{Field: "name", Operator: FilterOperatorLike, Value: "alice", Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}

func BenchmarkFilterGetWhereClause_in_5(b *testing.B) {
	f := Filter{Field: "id", Operator: FilterOperatorIn, Value: []string{"a", "b", "c", "d", "e"}, Table: "users"}
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
	f := Filter{Field: "id", Operator: FilterOperatorIn, Value: vals, Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}

func BenchmarkFilterGetWhereClause_notEq(b *testing.B) {
	f := Filter{Field: "status", Operator: FilterOperatorNotEq, Value: "deleted", Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}

func BenchmarkFilterGroupGetWhereClause_3filters(b *testing.B) {
	fg := FilterGroup{
		Operator: FilterGroupOperatorAnd,
		Filters: []any{
			Filter{Field: "id", Operator: FilterOperatorEq, Value: "123", Table: "users"},
			Filter{Field: "status", Operator: FilterOperatorNotEq, Value: "deleted", Table: "users"},
			Filter{Field: "name", Operator: FilterOperatorLike, Value: "alice", Table: "users"},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		fg.GetWhereClause()
	}
}

func BenchmarkFilterIsNull(b *testing.B) {
	f := Filter{Field: "deleted_at", Operator: FilterIsNull, Table: "users"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		f.GetWhereClause()
	}
}
