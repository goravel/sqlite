package sqlite

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cast"

	contractsschema "github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/contracts/log"
	"github.com/goravel/framework/database/schema"
	"github.com/goravel/framework/support/collect"
)

var _ contractsschema.Grammar = &Grammar{}

type Grammar struct {
	attributeCommands []string
	log               log.Log
	modifiers         []func(contractsschema.Blueprint, contractsschema.ColumnDefinition) string
	prefix            string
	serials           []string
	wrap              *schema.Wrap
}

func NewGrammar(log log.Log, prefix string) *Grammar {
	grammar := &Grammar{
		attributeCommands: []string{},
		log:               log,
		prefix:            prefix,
		serials:           []string{"bigInteger", "integer", "mediumInteger", "smallInteger", "tinyInteger"},
		wrap:              schema.NewWrap(prefix),
	}
	grammar.modifiers = []func(contractsschema.Blueprint, contractsschema.ColumnDefinition) string{
		grammar.ModifyDefault,
		grammar.ModifyIncrement,
		grammar.ModifyNullable,
	}

	return grammar
}
func (r *Grammar) CompileAdd(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return fmt.Sprintf("alter table %s add column %s", r.wrap.Table(blueprint.GetTableName()), r.getColumn(blueprint, command.Column))
}

func (r *Grammar) CompileChange(blueprint contractsschema.Blueprint, command *contractsschema.Command) []string {
	return nil
}

func (r *Grammar) CompileColumns(_, table string) (string, error) {
	table = r.prefix + table

	return fmt.Sprintf(
		`select name, type, not "notnull" as "nullable", dflt_value as "default", pk as "primary", hidden as "extra" `+
			"from pragma_table_xinfo(%s) order by cid asc", r.wrap.Quote(strings.ReplaceAll(table, ".", "__"))), nil
}

func (r *Grammar) CompileDefault(_ contractsschema.Blueprint, _ *contractsschema.Command) string {
	return ""
}

func (r *Grammar) CompileComment(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return ""
}

func (r *Grammar) CompileCreate(blueprint contractsschema.Blueprint) string {
	return fmt.Sprintf("create table %s (%s%s%s)",
		r.wrap.Table(blueprint.GetTableName()),
		strings.Join(r.getColumns(blueprint), ", "),
		r.addForeignKeys(getCommandsByName(blueprint.GetCommands(), "foreign")),
		r.addPrimaryKeys(getCommandByName(blueprint.GetCommands(), "primary")))
}

func (r *Grammar) CompileDisableWriteableSchema() string {
	return r.pragma("writable_schema", "0")
}

func (r *Grammar) CompileDrop(blueprint contractsschema.Blueprint) string {
	return fmt.Sprintf("drop table %s", r.wrap.Table(blueprint.GetTableName()))
}

func (r *Grammar) CompileDropAllDomains(domains []string) string {
	return ""
}

func (r *Grammar) CompileDropAllTables(_ string, _ []contractsschema.Table) []string {
	return []string{
		r.CompileEnableWriteableSchema(),
		"delete from sqlite_master where type in ('table', 'index', 'trigger')",
		r.CompileDisableWriteableSchema(),
	}
}

func (r *Grammar) CompileDropAllTypes(_ string, _ []contractsschema.Type) []string {
	return nil
}

func (r *Grammar) CompileDropAllViews(_ string, _ []contractsschema.View) []string {
	return []string{
		r.CompileEnableWriteableSchema(),
		"delete from sqlite_master where type in ('view')",
		r.CompileDisableWriteableSchema(),
	}
}

func (r *Grammar) CompileDropColumn(blueprint contractsschema.Blueprint, command *contractsschema.Command) []string {
	// TODO check Sqlite 3.35
	table := r.wrap.Table(blueprint.GetTableName())
	columns := r.wrap.PrefixArray("drop column", r.wrap.Columns(command.Columns))

	return collect.Map(columns, func(column string, _ int) string {
		return fmt.Sprintf("alter table %s %s", table, column)
	})
}

func (r *Grammar) CompileDropForeign(_ contractsschema.Blueprint, _ *contractsschema.Command) string {
	return ""
}

func (r *Grammar) CompileDropFullText(_ contractsschema.Blueprint, _ *contractsschema.Command) string {
	return ""
}

func (r *Grammar) CompileDropIfExists(blueprint contractsschema.Blueprint) string {
	return fmt.Sprintf("drop table if exists %s", r.wrap.Table(blueprint.GetTableName()))
}

func (r *Grammar) CompileDropIndex(_ contractsschema.Blueprint, command *contractsschema.Command) string {
	return fmt.Sprintf("drop index %s", r.wrap.Column(command.Index))
}

func (r *Grammar) CompileDropPrimary(_ contractsschema.Blueprint, _ *contractsschema.Command) string {
	return ""
}

func (r *Grammar) CompileDropUnique(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return r.CompileDropIndex(blueprint, command)
}

func (r *Grammar) CompileEnableWriteableSchema() string {
	return r.pragma("writable_schema", "1")
}

func (r *Grammar) CompileForeign(_ contractsschema.Blueprint, _ *contractsschema.Command) string {
	return ""
}

func (r *Grammar) CompileForeignKeys(_, table string) string {
	return fmt.Sprintf(
		`SELECT 
			GROUP_CONCAT("from") AS columns, 
			"table" AS foreign_table, 
			GROUP_CONCAT("to") AS foreign_columns, 
			on_update, 
			on_delete 
		FROM (
			SELECT * FROM pragma_foreign_key_list(%s) 
			ORDER BY id DESC, seq
		) 
		GROUP BY id, "table", on_update, on_delete`,
		r.wrap.Quote(strings.ReplaceAll(table, ".", "__")),
	)
}

func (r *Grammar) CompileFullText(_ contractsschema.Blueprint, _ *contractsschema.Command) string {
	return ""
}

func (r *Grammar) CompileIndex(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return fmt.Sprintf("create index %s on %s (%s)",
		r.wrap.Column(command.Index),
		r.wrap.Table(blueprint.GetTableName()),
		r.wrap.Columnize(command.Columns),
	)
}

func (r *Grammar) CompileIndexes(_, table string) (string, error) {
	table = r.prefix + table
	quotedTable := r.wrap.Quote(strings.ReplaceAll(table, ".", "__"))

	return fmt.Sprintf(
		`select 'primary' as name, group_concat(col) as columns, 1 as "unique", 1 as "primary" `+
			`from (select name as col from pragma_table_info(%s) where pk > 0 order by pk, cid) group by name `+
			`union select name, group_concat(col) as columns, "unique", origin = 'pk' as "primary" `+
			`from (select il.*, ii.name as col from pragma_index_list(%s) il, pragma_index_info(il.name) ii order by il.seq, ii.seqno) `+
			`group by name, "unique", "primary"`,
		quotedTable,
		r.wrap.Quote(table),
	), nil
}

func (r *Grammar) CompilePrimary(_ contractsschema.Blueprint, _ *contractsschema.Command) string {
	return ""
}

func (r *Grammar) CompileRebuild() string {
	return "vacuum"
}

func (r *Grammar) CompileRename(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return fmt.Sprintf("alter table %s rename to %s", r.wrap.Table(blueprint.GetTableName()), r.wrap.Table(command.To))
}

func (r *Grammar) CompileRenameIndex(s contractsschema.Schema, blueprint contractsschema.Blueprint, command *contractsschema.Command) []string {
	indexes, err := s.GetIndexes(blueprint.GetTableName())
	if err != nil {
		r.log.Errorf("failed to get %s indexes: %v", blueprint.GetTableName(), err)
		return nil
	}

	indexes = collect.Filter(indexes, func(index contractsschema.Index, _ int) bool {
		return index.Name == command.From
	})

	if len(indexes) == 0 {
		r.log.Warningf("index %s does not exist", command.From)
		return nil
	}
	if indexes[0].Primary {
		r.log.Warning("SQLite does not support altering primary keys")
		return nil
	}
	if indexes[0].Unique {
		return []string{
			r.CompileDropUnique(blueprint, &contractsschema.Command{
				Index: indexes[0].Name,
			}),
			r.CompileUnique(blueprint, &contractsschema.Command{
				Index:   command.To,
				Columns: indexes[0].Columns,
			}),
		}
	}

	return []string{
		r.CompileDropIndex(blueprint, &contractsschema.Command{
			Index: indexes[0].Name,
		}),
		r.CompileIndex(blueprint, &contractsschema.Command{
			Index:   command.To,
			Columns: indexes[0].Columns,
		}),
	}
}

func (r *Grammar) CompileTables(database string) string {
	return "select name from sqlite_master where type = 'table' and name not like 'sqlite_%' order by name"
}

func (r *Grammar) CompileTypes() string {
	return ""
}

func (r *Grammar) CompileUnique(blueprint contractsschema.Blueprint, command *contractsschema.Command) string {
	return fmt.Sprintf("create unique index %s on %s (%s)",
		r.wrap.Column(command.Index),
		r.wrap.Table(blueprint.GetTableName()),
		r.wrap.Columnize(command.Columns))
}

func (r *Grammar) CompileViews(database string) string {
	return "select name, sql as definition from sqlite_master where type = 'view' order by name"
}

func (r *Grammar) GetAttributeCommands() []string {
	return r.attributeCommands
}

func (r *Grammar) GetModifiers() []func(blueprint contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	return r.modifiers
}

func (r *Grammar) ModifyDefault(blueprint contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	if column.GetDefault() != nil {
		return fmt.Sprintf(" default %s", schema.ColumnDefaultValue(column.GetDefault()))
	}

	return ""
}

func (r *Grammar) ModifyNullable(blueprint contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	if column.GetNullable() {
		return " null"
	} else {
		return " not null"
	}
}

func (r *Grammar) ModifyIncrement(blueprint contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	if slices.Contains(r.serials, column.GetType()) && column.GetAutoIncrement() {
		return " primary key autoincrement"
	}

	return ""
}

func (r *Grammar) TypeBigInteger(column contractsschema.ColumnDefinition) string {
	return "integer"
}

func (r *Grammar) TypeBoolean(_ contractsschema.ColumnDefinition) string {
	return "tinyint(1)"
}

func (r *Grammar) TypeChar(column contractsschema.ColumnDefinition) string {
	return "varchar"
}

func (r *Grammar) TypeDate(column contractsschema.ColumnDefinition) string {
	return "date"
}

func (r *Grammar) TypeDateTime(column contractsschema.ColumnDefinition) string {
	return r.TypeTimestamp(column)
}

func (r *Grammar) TypeDateTimeTz(column contractsschema.ColumnDefinition) string {
	return r.TypeDateTime(column)
}

func (r *Grammar) TypeDecimal(column contractsschema.ColumnDefinition) string {
	return "numeric"
}

func (r *Grammar) TypeDouble(column contractsschema.ColumnDefinition) string {
	return "double"
}

func (r *Grammar) TypeEnum(column contractsschema.ColumnDefinition) string {
	return fmt.Sprintf(`varchar check ("%s" in (%s))`, column.GetName(), strings.Join(r.wrap.Quotes(cast.ToStringSlice(column.GetAllowed())), ", "))
}

func (r *Grammar) TypeFloat(column contractsschema.ColumnDefinition) string {
	return "float"
}

func (r *Grammar) TypeInteger(column contractsschema.ColumnDefinition) string {
	return "integer"
}

func (r *Grammar) TypeJson(column contractsschema.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeJsonb(column contractsschema.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeLongText(column contractsschema.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeMediumInteger(column contractsschema.ColumnDefinition) string {
	return "integer"
}

func (r *Grammar) TypeMediumText(column contractsschema.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeSmallInteger(column contractsschema.ColumnDefinition) string {
	return "integer"
}

func (r *Grammar) TypeString(column contractsschema.ColumnDefinition) string {
	return "varchar"
}

func (r *Grammar) TypeText(column contractsschema.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeTime(column contractsschema.ColumnDefinition) string {
	return "time"
}

func (r *Grammar) TypeTimeTz(column contractsschema.ColumnDefinition) string {
	return r.TypeTime(column)
}

func (r *Grammar) TypeTimestamp(column contractsschema.ColumnDefinition) string {
	if column.GetUseCurrent() {
		column.Default(schema.Expression("CURRENT_TIMESTAMP"))
	}

	return "datetime"
}

func (r *Grammar) TypeTimestampTz(column contractsschema.ColumnDefinition) string {
	return r.TypeTimestamp(column)
}

func (r *Grammar) TypeTinyInteger(column contractsschema.ColumnDefinition) string {
	return "integer"
}

func (r *Grammar) TypeTinyText(column contractsschema.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) addForeignKeys(commands []*contractsschema.Command) string {
	var sql string

	for _, command := range commands {
		sql += r.getForeignKey(command)
	}

	return sql
}

func (r *Grammar) addPrimaryKeys(command *contractsschema.Command) string {
	if command == nil {
		return ""
	}

	return fmt.Sprintf(", primary key (%s)", r.wrap.Columnize(command.Columns))
}

func (r *Grammar) getColumns(blueprint contractsschema.Blueprint) []string {
	var columns []string
	for _, column := range blueprint.GetAddedColumns() {
		columns = append(columns, r.getColumn(blueprint, column))
	}

	return columns
}

func (r *Grammar) getColumn(blueprint contractsschema.Blueprint, column contractsschema.ColumnDefinition) string {
	sql := fmt.Sprintf("%s %s", r.wrap.Column(column.GetName()), schema.ColumnType(r, column))

	for _, modifier := range r.modifiers {
		sql += modifier(blueprint, column)
	}

	return sql
}

func (r *Grammar) getForeignKey(command *contractsschema.Command) string {
	sql := fmt.Sprintf(", foreign key(%s) references %s(%s)",
		r.wrap.Columnize(command.Columns),
		r.wrap.Table(command.On),
		r.wrap.Columnize(command.References))

	if command.OnDelete != "" {
		sql += " on delete " + command.OnDelete
	}
	if command.OnUpdate != "" {
		sql += " on update " + command.OnUpdate
	}

	return sql
}

func (r *Grammar) pragma(name, value string) string {
	return fmt.Sprintf("pragma %s = %s", name, value)
}

func getCommandByName(commands []*contractsschema.Command, name string) *contractsschema.Command {
	commands = getCommandsByName(commands, name)
	if len(commands) == 0 {
		return nil
	}

	return commands[0]
}

func getCommandsByName(commands []*contractsschema.Command, name string) []*contractsschema.Command {
	var filteredCommands []*contractsschema.Command
	for _, command := range commands {
		if command.Name == name {
			filteredCommands = append(filteredCommands, command)
		}
	}

	return filteredCommands
}
