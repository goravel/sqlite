package sqlite

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/goravel/framework/contracts/database/schema"
)

type ProcessorTestSuite struct {
	suite.Suite
	processor *Processor
}

func TestProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessorTestSuite))
}

func (s *ProcessorTestSuite) SetupTest() {
	s.processor = NewProcessor()
}

func (s *ProcessorTestSuite) TestProcessColumns() {
	tests := []struct {
		name      string
		dbColumns []schema.DBColumn
		expected  []schema.Column
	}{
		{
			name: "ValidInput",
			dbColumns: []schema.DBColumn{
				{Name: "id", Type: "integer", Nullable: "false", Primary: true, Default: "1"},
				{Name: "name", Type: "varchar", Nullable: "true", Default: "default_name"},
			},
			expected: []schema.Column{
				{Autoincrement: true, Default: "1", Name: "id", Nullable: false, Type: "integer", TypeName: "integer"},
				{Autoincrement: false, Default: "default_name", Name: "name", Nullable: true, Type: "varchar", TypeName: "varchar"},
			},
		},
		{
			name:      "EmptyInput",
			dbColumns: []schema.DBColumn{},
		},
		{
			name: "NullableColumn",
			dbColumns: []schema.DBColumn{
				{Name: "description", Type: "text", Nullable: "true", Default: "default_description"},
			},
			expected: []schema.Column{
				{Autoincrement: false, Default: "default_description", Name: "description", Nullable: true, Type: "text", TypeName: "text"},
			},
		},
		{
			name: "NonNullableColumn",
			dbColumns: []schema.DBColumn{
				{Name: "created_at", Type: "timestamp", Nullable: "false", Default: "CURRENT_TIMESTAMP"},
			},
			expected: []schema.Column{
				{Autoincrement: false, Default: "CURRENT_TIMESTAMP", Name: "created_at", Nullable: false, Type: "timestamp", TypeName: "timestamp"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := s.processor.ProcessColumns(tt.dbColumns)
			s.Equal(tt.expected, result)
		})
	}
}

func (s *ProcessorTestSuite) TestProcessForeignKeys() {
	tests := []struct {
		name          string
		dbForeignKeys []schema.DBForeignKey
		expected      []schema.ForeignKey
	}{
		{
			name: "ValidInput",
			dbForeignKeys: []schema.DBForeignKey{
				{Name: "fk_user_id", Columns: "user_id", ForeignTable: "users", ForeignColumns: "id", OnUpdate: "CASCADE", OnDelete: "SET NULL"},
			},
			expected: []schema.ForeignKey{
				{Name: "fk_user_id", Columns: []string{"user_id"}, ForeignTable: "users", ForeignColumns: []string{"id"}, OnUpdate: "cascade", OnDelete: "set null"},
			},
		},
		{
			name:          "EmptyInput",
			dbForeignKeys: []schema.DBForeignKey{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := s.processor.ProcessForeignKeys(tt.dbForeignKeys)
			s.Equal(tt.expected, result)
		})
	}
}

func (s *ProcessorTestSuite) TestProcessIndexes() {
	// Test with valid indexes
	input := []schema.DBIndex{
		{Name: "INDEX_A", Type: "BTREE", Columns: "a,b"},
		{Name: "INDEX_B", Type: "HASH", Columns: "c,d"},
		{Name: "INDEX_C", Type: "HASH", Columns: "e,f", Primary: true},
	}
	expected := []schema.Index{
		{Name: "index_a", Columns: []string{"a", "b"}},
		{Name: "index_b", Columns: []string{"c", "d"}},
		{Name: "index_c", Columns: []string{"e", "f"}, Primary: true},
	}

	result := s.processor.ProcessIndexes(input)

	s.Equal(expected, result)

	// Test with valid indexes with multiple primary keys
	input = []schema.DBIndex{
		{Name: "INDEX_A", Type: "BTREE", Columns: "a,b"},
		{Name: "INDEX_B", Type: "HASH", Columns: "c,d"},
		{Name: "INDEX_C", Type: "HASH", Columns: "e,f", Primary: true},
		{Name: "INDEX_D", Type: "HASH", Columns: "g,h", Primary: true},
	}
	expected = []schema.Index{
		{Name: "index_a", Columns: []string{"a", "b"}},
		{Name: "index_b", Columns: []string{"c", "d"}},
	}

	result = s.processor.ProcessIndexes(input)

	s.Equal(expected, result)

	// Test with empty input
	input = []schema.DBIndex{}

	result = s.processor.ProcessIndexes(input)

	s.Nil(result)
}
