package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	contractsschema "github.com/goravel/framework/contracts/database/schema"
	mocksschema "github.com/goravel/framework/mocks/database/schema"
	mockslog "github.com/goravel/framework/mocks/log"
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
	commands := []*contractsschema.Command{
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
	mockBlueprint := mocksschema.NewBlueprint(s.T())
	mockColumn := mocksschema.NewColumnDefinition(s.T())

	mockBlueprint.EXPECT().GetTableName().Return("users").Once()
	mockColumn.EXPECT().GetName().Return("name").Once()
	mockColumn.EXPECT().GetType().Return("string").Twice()
	mockColumn.EXPECT().GetDefault().Return("goravel").Twice()
	mockColumn.EXPECT().GetNullable().Return(false).Once()

	sql := s.grammar.CompileAdd(mockBlueprint, &contractsschema.Command{
		Column: mockColumn,
	})

	s.Equal(`alter table "goravel_users" add column "name" varchar default 'goravel' not null`, sql)
}

func (s *GrammarSuite) TestCompileCreate() {
	mockColumn1 := mocksschema.NewColumnDefinition(s.T())
	mockColumn2 := mocksschema.NewColumnDefinition(s.T())
	mockBlueprint := mocksschema.NewBlueprint(s.T())

	// sqlite.go::CompileCreate
	mockBlueprint.EXPECT().GetTableName().Return("users").Once()
	// utils.go::getColumns
	mockBlueprint.EXPECT().GetAddedColumns().Return([]contractsschema.ColumnDefinition{
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
	mockBlueprint.EXPECT().GetCommands().Return([]*contractsschema.Command{
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
	mockBlueprint := mocksschema.NewBlueprint(s.T())
	mockBlueprint.EXPECT().GetTableName().Return("users").Once()

	s.Equal([]string{
		`alter table "goravel_users" drop column "id"`,
		`alter table "goravel_users" drop column "name"`,
	}, s.grammar.CompileDropColumn(mockBlueprint, &contractsschema.Command{
		Name:    "name",
		Columns: []string{"id", "name"},
	}))
}

func (s *GrammarSuite) TestCompileDropIfExists() {
	mockBlueprint := mocksschema.NewBlueprint(s.T())
	mockBlueprint.EXPECT().GetTableName().Return("users").Once()

	s.Equal(`drop table if exists "goravel_users"`, s.grammar.CompileDropIfExists(mockBlueprint))
}

func (s *GrammarSuite) TestCompileIndex() {
	mockBlueprint := mocksschema.NewBlueprint(s.T())
	mockBlueprint.EXPECT().GetTableName().Return("users").Once()
	command := &contractsschema.Command{
		Index:   "users",
		Columns: []string{"role_id", "permission_id"},
	}

	s.Equal(`create index "users" on "goravel_users" ("role_id", "permission_id")`, s.grammar.CompileIndex(mockBlueprint, command))
}

func (s *GrammarSuite) TestCompileRenameColumn() {
	mockBlueprint := mocksschema.NewBlueprint(s.T())
	mockColumn := mocksschema.NewColumnDefinition(s.T())

	mockBlueprint.EXPECT().GetTableName().Return("users").Once()

	sql, err := s.grammar.CompileRenameColumn(nil, mockBlueprint, &contractsschema.Command{
		Column: mockColumn,
		From:   "before",
		To:     "after",
	})

	s.NoError(err)
	s.Equal(`alter table "goravel_users" rename column "before" to "after"`, sql)
}

func (s *GrammarSuite) TestCompileRenameIndex() {
	var (
		mockSchema    *mocksschema.Schema
		mockBlueprint *mocksschema.Blueprint
	)

	beforeEach := func() {
		mockSchema = mocksschema.NewSchema(s.T())
		mockBlueprint = mocksschema.NewBlueprint(s.T())
	}

	tests := []struct {
		name    string
		command *contractsschema.Command
		setup   func()
		expect  []string
	}{
		{
			name: "failed to get indexes",
			setup: func() {
				tableName := "users"
				mockBlueprint.EXPECT().GetTableName().Return(tableName).Twice()
				mockSchema.EXPECT().GetIndexes(tableName).Return(nil, assert.AnError).Once()
				s.mockLog.EXPECT().Errorf("failed to get %s indexes: %v", tableName, assert.AnError).Once()
			},
		},
		{
			name: "index does not exist",
			command: &contractsschema.Command{
				From: "users",
			},
			setup: func() {
				tableName := "users"
				mockBlueprint.EXPECT().GetTableName().Return(tableName).Once()
				mockSchema.EXPECT().GetIndexes(tableName).Return([]contractsschema.Index{
					{
						Name: "admins",
					},
				}, nil).Once()
				s.mockLog.EXPECT().Warningf("index %s does not exist", "users").Once()
			},
		},
		{
			name: "index is primary",
			command: &contractsschema.Command{
				From: "users",
			},
			setup: func() {
				tableName := "users"
				mockBlueprint.EXPECT().GetTableName().Return(tableName).Once()
				mockSchema.EXPECT().GetIndexes(tableName).Return([]contractsschema.Index{
					{
						Name:    "users",
						Primary: true,
					},
				}, nil).Once()
				s.mockLog.EXPECT().Warning("SQLite does not support altering primary keys").Once()
			},
		},
		{
			name: "index is unique",
			command: &contractsschema.Command{
				From: "users",
				To:   "admins",
			},
			setup: func() {
				tableName := "users"
				mockBlueprint.EXPECT().GetTableName().Return(tableName).Twice()
				mockSchema.EXPECT().GetIndexes(tableName).Return([]contractsschema.Index{
					{
						Columns: []string{"role_id", "permission_id"},
						Name:    "users",
						Unique:  true,
					},
				}, nil).Once()
			},
			expect: []string{
				`drop index "users"`,
				`create unique index "admins" on "goravel_users" ("role_id", "permission_id")`,
			},
		},
		{
			name: "success",
			command: &contractsschema.Command{
				From: "users",
				To:   "admins",
			},
			setup: func() {
				tableName := "users"
				mockBlueprint.EXPECT().GetTableName().Return(tableName).Twice()
				mockSchema.EXPECT().GetIndexes(tableName).Return([]contractsschema.Index{
					{
						Columns: []string{"role_id", "permission_id"},
						Name:    "users",
					},
				}, nil).Once()
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

			s.Equal(test.expect, s.grammar.CompileRenameIndex(mockSchema, mockBlueprint, test.command))
		})
	}
}

func (s *GrammarSuite) TestGetColumns() {
	mockColumn1 := mocksschema.NewColumnDefinition(s.T())
	mockColumn2 := mocksschema.NewColumnDefinition(s.T())
	mockBlueprint := mocksschema.NewBlueprint(s.T())

	mockBlueprint.EXPECT().GetAddedColumns().Return([]contractsschema.ColumnDefinition{
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
		mockBlueprint *mocksschema.Blueprint
		mockColumn    *mocksschema.ColumnDefinition
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
			mockBlueprint = mocksschema.NewBlueprint(s.T())
			mockColumn = mocksschema.NewColumnDefinition(s.T())

			test.setup()

			sql := s.grammar.ModifyDefault(mockBlueprint, mockColumn)

			s.Equal(test.expectSql, sql)
		})
	}
}

func (s *GrammarSuite) TestModifyNullable() {
	mockBlueprint := mocksschema.NewBlueprint(s.T())

	mockColumn := mocksschema.NewColumnDefinition(s.T())

	mockColumn.EXPECT().GetNullable().Return(true).Once()

	s.Equal(" null", s.grammar.ModifyNullable(mockBlueprint, mockColumn))

	mockColumn.EXPECT().GetNullable().Return(false).Once()

	s.Equal(" not null", s.grammar.ModifyNullable(mockBlueprint, mockColumn))
}

func (s *GrammarSuite) TestModifyIncrement() {
	mockBlueprint := mocksschema.NewBlueprint(s.T())

	mockColumn := mocksschema.NewColumnDefinition(s.T())
	mockColumn.EXPECT().GetType().Return("bigInteger").Once()
	mockColumn.EXPECT().GetAutoIncrement().Return(true).Once()

	s.Equal(" primary key autoincrement", s.grammar.ModifyIncrement(mockBlueprint, mockColumn))
}

func (s *GrammarSuite) TestTypeBoolean() {
	mockColumn := mocksschema.NewColumnDefinition(s.T())

	s.Equal("tinyint(1)", s.grammar.TypeBoolean(mockColumn))
}

func (s *GrammarSuite) TestTypeEnum() {
	mockColumn := mocksschema.NewColumnDefinition(s.T())
	mockColumn.EXPECT().GetName().Return("a").Once()
	mockColumn.EXPECT().GetAllowed().Return([]any{"a", "b"}).Once()

	s.Equal(`varchar check ("a" in ('a', 'b'))`, s.grammar.TypeEnum(mockColumn))
}

func TestGetCommandByName(t *testing.T) {
	commands := []*contractsschema.Command{
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
