package sqlite

import (
	"testing"

	contractsdriver "github.com/goravel/framework/contracts/database/driver"
	mocksdriver "github.com/goravel/framework/mocks/database/driver"
	mockslog "github.com/goravel/framework/mocks/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GrammarSuite struct {
	suite.Suite
	grammar *Grammar
	mockLog *mockslog.Log
}

func TestGrammarSuite(t *testing.T) {
	suite.Run(t, &GrammarSuite{})
}

func (s *GrammarSuite) SetupTest() {
	s.mockLog = mockslog.NewLog(s.T())
	s.grammar = NewGrammar(s.mockLog, "goravel_")
}

func (s *GrammarSuite) TestAddForeignKeys() {
	commands := []*contractsdriver.Command{
		{
			Columns:    []string{"role_id", "permission_id"},
			On:         "roles",
			References: []string{"id", "user_id"},
			OnDelete:   "cascade",
			OnUpdate:   "restrict",
		},
		{
			Columns:    []string{"permission_id", "role_id"},
			On:         "permissions",
			References: []string{"id", "user_id"},
		},
	}

	s.Equal(`, foreign key("role_id", "permission_id") references "goravel_roles"("id", "user_id") on delete cascade on update restrict, foreign key("permission_id", "role_id") references "goravel_permissions"("id", "user_id")`, s.grammar.addForeignKeys(commands))
}

func (s *GrammarSuite) TestCompileAdd() {
	mockBlueprint := mocksdriver.NewBlueprint(s.T())
	mockColumn := mocksdriver.NewColumnDefinition(s.T())

	mockBlueprint.EXPECT().GetTableName().Return("users").Once()
	mockColumn.EXPECT().GetName().Return("name").Once()
	mockColumn.EXPECT().GetType().Return("string").Twice()
	mockColumn.EXPECT().GetDefault().Return("goravel").Twice()
	mockColumn.EXPECT().GetNullable().Return(false).Once()

	sql := s.grammar.CompileAdd(mockBlueprint, &contractsdriver.Command{
		Column: mockColumn,
	})

	s.Equal(`alter table "goravel_users" add column "name" varchar default 'goravel' not null`, sql)
}

func (s *GrammarSuite) TestCompileCreate() {
	mockColumn1 := mocksdriver.NewColumnDefinition(s.T())
	mockColumn2 := mocksdriver.NewColumnDefinition(s.T())
	mockBlueprint := mocksdriver.NewBlueprint(s.T())

	// sqlite.go::CompileCreate
	mockBlueprint.EXPECT().GetTableName().Return("users").Once()
	// utils.go::getColumns
	mockBlueprint.EXPECT().GetAddedColumns().Return([]contractsdriver.ColumnDefinition{
		mockColumn1, mockColumn2,
	}).Once()
	// utils.go::getColumns
	mockColumn1.EXPECT().GetName().Return("id").Once()
	// utils.go::getType
	mockColumn1.EXPECT().GetType().Return("integer").Once()
	// sqlite.go::TypeInteger
	mockColumn1.EXPECT().GetAutoIncrement().Return(true).Once()
	// sqlite.go::ModifyDefault
	mockColumn1.EXPECT().GetDefault().Return(nil).Once()
	// sqlite.go::ModifyIncrement
	mockColumn1.EXPECT().GetType().Return("integer").Once()
	// sqlite.go::ModifyNullable
	mockColumn1.EXPECT().GetNullable().Return(false).Once()

	// utils.go::getColumns
	mockColumn2.EXPECT().GetName().Return("name").Once()
	// utils.go::getType
	mockColumn2.EXPECT().GetType().Return("string").Once()
	// sqlite.go::ModifyDefault
	mockColumn2.EXPECT().GetDefault().Return(nil).Once()
	// sqlite.go::ModifyIncrement
	mockColumn2.EXPECT().GetType().Return("string").Once()
	// sqlite.go::ModifyNullable
	mockColumn2.EXPECT().GetNullable().Return(true).Once()

	// sqlite.go::CompileCreate
	mockBlueprint.EXPECT().GetCommands().Return([]*contractsdriver.Command{
		{
			Name:    "primary",
			Columns: []string{"id"},
		},
		{
			Name:       "foreign",
			Columns:    []string{"role_id", "permission_id"},
			On:         "roles",
			References: []string{"id"},
			OnDelete:   "cascade",
			OnUpdate:   "restrict",
		},
		{
			Name:       "foreign",
			Columns:    []string{"permission_id", "role_id"},
			On:         "permissions",
			References: []string{"id"},
			OnDelete:   "cascade",
			OnUpdate:   "restrict",
		},
	}).Twice()

	s.Equal(`create table "goravel_users" ("id" integer primary key autoincrement not null, "name" varchar null, foreign key("role_id", "permission_id") references "goravel_roles"("id") on delete cascade on update restrict, foreign key("permission_id", "role_id") references "goravel_permissions"("id") on delete cascade on update restrict, primary key ("id"))`,
		s.grammar.CompileCreate(mockBlueprint))
}

func (s *GrammarSuite) TestCompileDropColumn() {
	mockBlueprint := mocksdriver.NewBlueprint(s.T())
	mockBlueprint.EXPECT().GetTableName().Return("users").Once()

	s.Equal([]string{
		`alter table "goravel_users" drop column "id"`,
		`alter table "goravel_users" drop column "name"`,
	}, s.grammar.CompileDropColumn(mockBlueprint, &contractsdriver.Command{
		Name:    "name",
		Columns: []string{"id", "name"},
	}))
}

func (s *GrammarSuite) TestCompileDropIfExists() {
	mockBlueprint := mocksdriver.NewBlueprint(s.T())
	mockBlueprint.EXPECT().GetTableName().Return("users").Once()

	s.Equal(`drop table if exists "goravel_users"`, s.grammar.CompileDropIfExists(mockBlueprint))
}

func (s *GrammarSuite) TestCompileIndex() {
	mockBlueprint := mocksdriver.NewBlueprint(s.T())
	mockBlueprint.EXPECT().GetTableName().Return("users").Once()
	command := &contractsdriver.Command{
		Index:   "users",
		Columns: []string{"role_id", "permission_id"},
	}

	s.Equal(`create index "users" on "goravel_users" ("role_id", "permission_id")`, s.grammar.CompileIndex(mockBlueprint, command))
}

func (s *GrammarSuite) TestCompileJsonContains() {
	tests := []struct {
		name          string
		column        string
		value         any
		isNot         bool
		expectedSql   string
		expectedValue []any
	}{
		{
			name:          "single path with single value",
			column:        "data->details",
			value:         "value1",
			expectedSql:   `exists (select 1 from json_each(json_extract("data", '$."details"')) where value = ? )`,
			expectedValue: []any{"value1"},
		},
		{
			name:          "single path with multiple values",
			column:        "data->details",
			value:         []string{"value1", "value2"},
			expectedSql:   `exists (select 1 from json_each(json_extract("data", '$."details"')) where value = ? ) AND exists (select 1 from json_each(json_extract("data", '$."details"')) where value = ? )`,
			expectedValue: []any{"value1", "value2"},
		},
		{
			name:          "nested path with single value",
			column:        "data->details->subdetails[0]",
			value:         "value1",
			expectedSql:   `exists (select 1 from json_each(json_extract("data", '$."details"."subdetails"[0]')) where value = ? )`,
			expectedValue: []any{"value1"},
		},
		{
			name:          "nested path with multiple values",
			column:        "data->details[0]->subdetails",
			value:         []string{"value1", "value2"},
			expectedSql:   `exists (select 1 from json_each(json_extract("data", '$."details"[0]."subdetails"')) where value = ? ) AND exists (select 1 from json_each(json_extract("data", '$."details"[0]."subdetails"')) where value = ? )`,
			expectedValue: []any{"value1", "value2"},
		},
		{
			name:          "with is not condition",
			column:        "data->details",
			value:         "value1",
			isNot:         true,
			expectedSql:   `not exists (select 1 from json_each(json_extract("data", '$."details"')) where value = ? )`,
			expectedValue: []any{"value1"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			actualSql, actualValue, err := s.grammar.CompileJsonContains(tt.column, tt.value, tt.isNot)
			s.Equal(tt.expectedSql, actualSql)
			s.Equal(tt.expectedValue, actualValue)
			s.NoError(err)
		})
	}
}

func (s *GrammarSuite) TestCompileJsonContainKey() {
	tests := []struct {
		name        string
		column      string
		isNot       bool
		expectedSql string
	}{
		{
			name:        "single path",
			column:      "data->details",
			expectedSql: `json_type("data", '$."details"') is not null`,
		},
		{
			name:        "single path with is not",
			column:      "data->details",
			isNot:       true,
			expectedSql: `not json_type("data", '$."details"') is not null`,
		},
		{
			name:        "nested path",
			column:      "data->details->subdetails",
			expectedSql: `json_type("data", '$."details"."subdetails"') is not null`,
		},
		{
			name:        "nested path with array index",
			column:      "data->details[0]->subdetails",
			expectedSql: `json_type("data", '$."details"[0]."subdetails"') is not null`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.expectedSql, s.grammar.CompileJsonContainsKey(tt.column, tt.isNot))
		})
	}
}

func (s *GrammarSuite) TestCompileJsonLength() {
	tests := []struct {
		name        string
		column      string
		expectedSql string
	}{
		{
			name:        "single path",
			column:      "data->details",
			expectedSql: `json_array_length("data", '$."details"')`,
		},
		{
			name:        "nested path",
			column:      "data->details->subdetails",
			expectedSql: `json_array_length("data", '$."details"."subdetails"')`,
		},
		{
			name:        "nested path with array index",
			column:      "data->details[0]->subdetails",
			expectedSql: `json_array_length("data", '$."details"[0]."subdetails"')`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.expectedSql, s.grammar.CompileJsonLength(tt.column))
		})
	}
}

func (s *GrammarSuite) TestCompileRenameColumn() {
	mockBlueprint := mocksdriver.NewBlueprint(s.T())
	mockColumn := mocksdriver.NewColumnDefinition(s.T())

	mockBlueprint.EXPECT().GetTableName().Return("users").Once()

	sql, err := s.grammar.CompileRenameColumn(mockBlueprint, &contractsdriver.Command{
		Column: mockColumn,
		From:   "before",
		To:     "after",
	}, nil)

	s.NoError(err)
	s.Equal(`alter table "goravel_users" rename column "before" to "after"`, sql)
}

func (s *GrammarSuite) TestCompileRenameIndex() {
	var (
		mockBlueprint *mocksdriver.Blueprint
	)

	beforeEach := func() {
		mockBlueprint = mocksdriver.NewBlueprint(s.T())
	}

	tests := []struct {
		name    string
		command *contractsdriver.Command
		indexes []contractsdriver.Index
		setup   func()
		expect  []string
	}{
		{
			name: "index does not exist",
			command: &contractsdriver.Command{
				From: "users",
			},
			indexes: []contractsdriver.Index{
				{
					Name: "admins",
				},
			},
			setup: func() {
				s.mockLog.EXPECT().Warningf("index %s does not exist", "users").Once()
			},
		},
		{
			name: "index is primary",
			command: &contractsdriver.Command{
				From: "users",
			},
			indexes: []contractsdriver.Index{
				{
					Name:    "users",
					Primary: true,
				},
			},
			setup: func() {
				s.mockLog.EXPECT().Warning("SQLite does not support altering primary keys").Once()
			},
		},
		{
			name: "index is unique",
			command: &contractsdriver.Command{
				From: "users",
				To:   "admins",
			},
			indexes: []contractsdriver.Index{
				{
					Columns: []string{"role_id", "permission_id"},
					Name:    "users",
					Unique:  true,
				},
			},
			setup: func() {
				mockBlueprint.EXPECT().GetTableName().Return("users").Once()
			},
			expect: []string{
				`drop index "users"`,
				`create unique index "admins" on "goravel_users" ("role_id", "permission_id")`,
			},
		},
		{
			name: "success",
			command: &contractsdriver.Command{
				From: "users",
				To:   "admins",
			},
			indexes: []contractsdriver.Index{
				{
					Columns: []string{"role_id", "permission_id"},
					Name:    "users",
				},
			},
			setup: func() {
				mockBlueprint.EXPECT().GetTableName().Return("users").Once()
			},
			expect: []string{
				`drop index "users"`,
				`create index "admins" on "goravel_users" ("role_id", "permission_id")`,
			},
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			beforeEach()
			test.setup()

			s.Equal(test.expect, s.grammar.CompileRenameIndex(mockBlueprint, test.command, test.indexes))
		})
	}
}

func (s *GrammarSuite) TestGetColumns() {
	mockColumn1 := mocksdriver.NewColumnDefinition(s.T())
	mockColumn2 := mocksdriver.NewColumnDefinition(s.T())
	mockBlueprint := mocksdriver.NewBlueprint(s.T())

	mockBlueprint.EXPECT().GetAddedColumns().Return([]contractsdriver.ColumnDefinition{
		mockColumn1, mockColumn2,
	}).Once()

	mockColumn1.EXPECT().GetName().Return("id").Once()
	mockColumn1.EXPECT().GetType().Return("integer").Twice()
	mockColumn1.EXPECT().GetDefault().Return(nil).Once()
	mockColumn1.EXPECT().GetNullable().Return(false).Once()
	mockColumn1.EXPECT().GetAutoIncrement().Return(true).Once()

	mockColumn2.EXPECT().GetName().Return("name").Once()
	mockColumn2.EXPECT().GetType().Return("string").Twice()
	mockColumn2.EXPECT().GetDefault().Return("goravel").Twice()
	mockColumn2.EXPECT().GetNullable().Return(true).Once()

	s.Equal([]string{"\"id\" integer primary key autoincrement not null", "\"name\" varchar default 'goravel' null"}, s.grammar.getColumns(mockBlueprint))
}

func (s *GrammarSuite) TestModifyDefault() {
	var (
		mockBlueprint *mocksdriver.Blueprint
		mockColumn    *mocksdriver.ColumnDefinition
	)

	tests := []struct {
		name      string
		setup     func()
		expectSql string
	}{
		{
			name: "without change and default is nil",
			setup: func() {
				mockColumn.EXPECT().GetDefault().Return(nil).Once()
			},
		},
		{
			name: "without change and default is not nil",
			setup: func() {
				mockColumn.EXPECT().GetDefault().Return("goravel").Twice()
			},
			expectSql: " default 'goravel'",
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			mockBlueprint = mocksdriver.NewBlueprint(s.T())
			mockColumn = mocksdriver.NewColumnDefinition(s.T())

			test.setup()

			sql := s.grammar.ModifyDefault(mockBlueprint, mockColumn)

			s.Equal(test.expectSql, sql)
		})
	}
}

func (s *GrammarSuite) TestModifyNullable() {
	mockBlueprint := mocksdriver.NewBlueprint(s.T())

	mockColumn := mocksdriver.NewColumnDefinition(s.T())

	mockColumn.EXPECT().GetNullable().Return(true).Once()

	s.Equal(" null", s.grammar.ModifyNullable(mockBlueprint, mockColumn))

	mockColumn.EXPECT().GetNullable().Return(false).Once()

	s.Equal(" not null", s.grammar.ModifyNullable(mockBlueprint, mockColumn))
}

func (s *GrammarSuite) TestModifyIncrement() {
	mockBlueprint := mocksdriver.NewBlueprint(s.T())

	mockColumn := mocksdriver.NewColumnDefinition(s.T())
	mockColumn.EXPECT().GetType().Return("bigInteger").Once()
	mockColumn.EXPECT().GetAutoIncrement().Return(true).Once()

	s.Equal(" primary key autoincrement", s.grammar.ModifyIncrement(mockBlueprint, mockColumn))
}

func (s *GrammarSuite) TestTypeBoolean() {
	mockColumn := mocksdriver.NewColumnDefinition(s.T())

	s.Equal("tinyint(1)", s.grammar.TypeBoolean(mockColumn))
}

func (s *GrammarSuite) TestTypeEnum() {
	mockColumn := mocksdriver.NewColumnDefinition(s.T())
	mockColumn.EXPECT().GetName().Return("a").Once()
	mockColumn.EXPECT().GetAllowed().Return([]any{"a", "b"}).Once()

	s.Equal(`varchar check ("a" in ('a', 'b'))`, s.grammar.TypeEnum(mockColumn))
}

func TestGetCommandByName(t *testing.T) {
	commands := []*contractsdriver.Command{
		{Name: "create"},
		{Name: "update"},
		{Name: "delete"},
	}

	// Test case: Command exists
	result := getCommandByName(commands, "update")
	assert.NotNil(t, result)
	assert.Equal(t, "update", result.Name)

	// Test case: Command does not exist
	result = getCommandByName(commands, "drop")
	assert.Nil(t, result)
}
