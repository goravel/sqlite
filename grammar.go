package sqlite

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/goravel/framework/contracts/database/driver"
	"github.com/goravel/framework/contracts/log"
	databasedb "github.com/goravel/framework/database/db"
	"github.com/goravel/framework/database/schema"
	"github.com/goravel/framework/support/collect"
	"github.com/spf13/cast"
	"gorm.io/gorm/clause"
)

var _ driver.Grammar = &Grammar{}

type Grammar struct {
	attributeCommands []string
	log               log.Log
	modifiers         []func(driver.Blueprint, driver.ColumnDefinition) string
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
	grammar.modifiers = []func(driver.Blueprint, driver.ColumnDefinition) string{
		grammar.ModifyDefault,
		grammar.ModifyIncrement,
		grammar.ModifyNullable,
	}

	return grammar
}
func (r *Grammar) CompileAdd(blueprint driver.Blueprint, command *driver.Command) string {
	return fmt.Sprintf("alter table %s add column %s", r.wrap.Table(blueprint.GetTableName()), r.getColumn(blueprint, command.Column))
}

func (r *Grammar) CompileChange(blueprint driver.Blueprint, command *driver.Command) []string {
	return nil
}

func (r *Grammar) CompileColumns(_, table string) (string, error) {
	table = r.prefix + table

	return fmt.Sprintf(
		`select name, type, not "notnull" as "nullable", dflt_value as "default", pk as "primary", hidden as "extra" `+
			"from pragma_table_xinfo(%s) order by cid asc", r.wrap.Quote(strings.ReplaceAll(table, ".", "__"))), nil
}

func (r *Grammar) CompileDefault(_ driver.Blueprint, _ *driver.Command) string {
	return ""
}

func (r *Grammar) CompileComment(blueprint driver.Blueprint, command *driver.Command) string {
	return ""
}

func (r *Grammar) CompileCreate(blueprint driver.Blueprint) string {
	return fmt.Sprintf("create table %s (%s%s%s)",
		r.wrap.Table(blueprint.GetTableName()),
		strings.Join(r.getColumns(blueprint), ", "),
		r.addForeignKeys(getCommandsByName(blueprint.GetCommands(), "foreign")),
		r.addPrimaryKeys(getCommandByName(blueprint.GetCommands(), "primary")))
}

func (r *Grammar) CompileDisableWriteableSchema() string {
	return r.pragma("writable_schema", "0")
}

func (r *Grammar) CompileDrop(blueprint driver.Blueprint) string {
	return fmt.Sprintf("drop table %s", r.wrap.Table(blueprint.GetTableName()))
}

func (r *Grammar) CompileDropAllDomains(domains []string) string {
	return ""
}

func (r *Grammar) CompileDropAllTables(_ string, _ []driver.Table) []string {
	return []string{
		r.CompileEnableWriteableSchema(),
		"delete from sqlite_master where type in ('table', 'index', 'trigger')",
		r.CompileDisableWriteableSchema(),
	}
}

func (r *Grammar) CompileDropAllTypes(_ string, _ []driver.Type) []string {
	return nil
}

func (r *Grammar) CompileDropAllViews(_ string, _ []driver.View) []string {
	return []string{
		r.CompileEnableWriteableSchema(),
		"delete from sqlite_master where type in ('view')",
		r.CompileDisableWriteableSchema(),
	}
}

func (r *Grammar) CompileDropColumn(blueprint driver.Blueprint, command *driver.Command) []string {
	// TODO check Sqlite 3.35
	table := r.wrap.Table(blueprint.GetTableName())
	columns := r.wrap.PrefixArray("drop column", r.wrap.Columns(command.Columns))

	return collect.Map(columns, func(column string, _ int) string {
		return fmt.Sprintf("alter table %s %s", table, column)
	})
}

func (r *Grammar) CompileDropForeign(_ driver.Blueprint, _ *driver.Command) string {
	return ""
}

func (r *Grammar) CompileDropFullText(_ driver.Blueprint, _ *driver.Command) string {
	return ""
}

func (r *Grammar) CompileDropIfExists(blueprint driver.Blueprint) string {
	return fmt.Sprintf("drop table if exists %s", r.wrap.Table(blueprint.GetTableName()))
}

func (r *Grammar) CompileDropIndex(_ driver.Blueprint, command *driver.Command) string {
	return fmt.Sprintf("drop index %s", r.wrap.Column(command.Index))
}

func (r *Grammar) CompileDropPrimary(_ driver.Blueprint, _ *driver.Command) string {
	return ""
}

func (r *Grammar) CompileDropUnique(blueprint driver.Blueprint, command *driver.Command) string {
	return r.CompileDropIndex(blueprint, command)
}

func (r *Grammar) CompileEnableWriteableSchema() string {
	return r.pragma("writable_schema", "1")
}

func (r *Grammar) CompileForeign(_ driver.Blueprint, _ *driver.Command) string {
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

func (r *Grammar) CompileFullText(_ driver.Blueprint, _ *driver.Command) string {
	return ""
}

func (r *Grammar) CompileIndex(blueprint driver.Blueprint, command *driver.Command) string {
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

func (r *Grammar) CompileJsonColumnsUpdate(values map[string]any) (map[string]any, error) {
	var (
		compiled = make(map[string]any)
		json     = App.GetJson()
	)
	for key, value := range values {
		if strings.Contains(key, "->") {
			segments := strings.SplitN(key, "->", 2)
			column, path := segments[0], strings.Trim(r.wrap.JsonPath(segments[1]), "'")

			val := reflect.ValueOf(value)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}

			if kind := val.Kind(); kind == reflect.Slice || kind == reflect.Array || kind == reflect.Map || kind == reflect.Struct || kind == reflect.Bool {
				binding, err := json.Marshal(value)
				if err != nil {
					return nil, err
				}
				value = databasedb.Raw("json(?)", string(binding))
			}

			expr, ok := compiled[column]
			if !ok {
				expr = databasedb.Raw(r.wrap.Column(column))
			}

			compiled[column] = databasedb.Raw("json_set(?,?,?)", expr, path, value)

			continue
		}

		compiled[key] = value
	}

	return compiled, nil
}

func (r *Grammar) CompileJsonContains(column string, value any, isNot bool) (string, []any, error) {
	field, path := r.wrap.JsonFieldAndPath(column)
	query := r.wrap.Not(fmt.Sprintf("exists (select 1 from json_each(json_extract(%s%s)) where value = ? )", field, path), isNot)

	if val := reflect.ValueOf(value); val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		values := make([]any, val.Len())
		queries := make([]string, val.Len())
		for i := 0; i < val.Len(); i++ {
			values[i] = val.Index(i).Interface()
			queries[i] = query
		}

		return strings.Join(queries, " AND "), values, nil
	}

	return query, []any{value}, nil
}

func (r *Grammar) CompileJsonContainsKey(column string, isNot bool) string {
	field, path := r.wrap.JsonFieldAndPath(column)

	return r.wrap.Not(fmt.Sprintf("json_type(%s%s) is not null", field, path), isNot)
}

func (r *Grammar) CompileJsonLength(column string) string {
	field, path := r.wrap.JsonFieldAndPath(column)

	return fmt.Sprintf("json_array_length(%s%s)", field, path)
}

func (r *Grammar) CompileJsonSelector(column string) string {
	field, path := r.wrap.JsonFieldAndPath(column)

	return fmt.Sprintf("json_extract(%s%s)", field, path)
}

func (r *Grammar) CompileJsonValues(args ...any) []any {

	return args
}

func (r *Grammar) CompileLockForUpdate(builder sq.SelectBuilder, conditions *driver.Conditions) sq.SelectBuilder {
	return builder
}

func (r *Grammar) CompileLockForUpdateForGorm() clause.Expression {
	return nil
}

func (r *Grammar) CompilePlaceholderFormat() driver.PlaceholderFormat {
	return nil
}

func (r *Grammar) CompilePrimary(_ driver.Blueprint, _ *driver.Command) string {
	return ""
}

func (r *Grammar) CompileInRandomOrder(builder sq.SelectBuilder, conditions *driver.Conditions) sq.SelectBuilder {
	if conditions.InRandomOrder != nil && *conditions.InRandomOrder {
		conditions.OrderBy = []string{"RANDOM()"}
	}

	return builder
}

func (r *Grammar) CompilePrune(_ string) string {
	return "vacuum"
}

func (r *Grammar) CompileRandomOrderForGorm() string {
	return "RANDOM()"
}

func (r *Grammar) CompileRename(blueprint driver.Blueprint, command *driver.Command) string {
	return fmt.Sprintf("alter table %s rename to %s", r.wrap.Table(blueprint.GetTableName()), r.wrap.Table(command.To))
}

func (r *Grammar) CompileRenameColumn(blueprint driver.Blueprint, command *driver.Command, _ []driver.Column) (string, error) {
	return fmt.Sprintf("alter table %s rename column %s to %s",
		r.wrap.Table(blueprint.GetTableName()),
		r.wrap.Column(command.From),
		r.wrap.Column(command.To),
	), nil
}

func (r *Grammar) CompileRenameIndex(blueprint driver.Blueprint, command *driver.Command, indexes []driver.Index) []string {
	indexes = collect.Filter(indexes, func(index driver.Index, _ int) bool {
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
			r.CompileDropUnique(blueprint, &driver.Command{
				Index: indexes[0].Name,
			}),
			r.CompileUnique(blueprint, &driver.Command{
				Index:   command.To,
				Columns: indexes[0].Columns,
			}),
		}
	}

	return []string{
		r.CompileDropIndex(blueprint, &driver.Command{
			Index: indexes[0].Name,
		}),
		r.CompileIndex(blueprint, &driver.Command{
			Index:   command.To,
			Columns: indexes[0].Columns,
		}),
	}
}

func (r *Grammar) CompileSharedLock(builder sq.SelectBuilder, conditions *driver.Conditions) sq.SelectBuilder {
	return builder
}

func (r *Grammar) CompileSharedLockForGorm() clause.Expression {
	return nil
}

func (r *Grammar) CompileTables(database string) string {
	return "select name from sqlite_master where type = 'table' and name not like 'sqlite_%' order by name"
}

func (r *Grammar) CompileTableComment(_ driver.Blueprint, _ *driver.Command) string {
	return ""
}

func (r *Grammar) CompileTypes() string {
	return ""
}

func (r *Grammar) CompileUnique(blueprint driver.Blueprint, command *driver.Command) string {
	return fmt.Sprintf("create unique index %s on %s (%s)",
		r.wrap.Column(command.Index),
		r.wrap.Table(blueprint.GetTableName()),
		r.wrap.Columnize(command.Columns))
}

func (r *Grammar) CompileVersion() string {
	return "SELECT sqlite_version() AS value;"
}

func (r *Grammar) CompileViews(database string) string {
	return "select name, sql as definition from sqlite_master where type = 'view' order by name"
}

func (r *Grammar) GetAttributeCommands() []string {
	return r.attributeCommands
}

func (r *Grammar) GetModifiers() []func(blueprint driver.Blueprint, column driver.ColumnDefinition) string {
	return r.modifiers
}

func (r *Grammar) ModifyDefault(blueprint driver.Blueprint, column driver.ColumnDefinition) string {
	if column.GetDefault() != nil {
		return fmt.Sprintf(" default %s", schema.ColumnDefaultValue(column.GetDefault()))
	}

	return ""
}

func (r *Grammar) ModifyNullable(blueprint driver.Blueprint, column driver.ColumnDefinition) string {
	if column.GetNullable() {
		return " null"
	} else {
		return " not null"
	}
}

func (r *Grammar) ModifyIncrement(blueprint driver.Blueprint, column driver.ColumnDefinition) string {
	if slices.Contains(r.serials, column.GetType()) && column.GetAutoIncrement() {
		return " primary key autoincrement"
	}

	return ""
}

func (r *Grammar) TypeBigInteger(column driver.ColumnDefinition) string {
	return "integer"
}

func (r *Grammar) TypeBoolean(_ driver.ColumnDefinition) string {
	return "tinyint(1)"
}

func (r *Grammar) TypeChar(column driver.ColumnDefinition) string {
	return "varchar"
}

func (r *Grammar) TypeDate(column driver.ColumnDefinition) string {
	return "date"
}

func (r *Grammar) TypeDateTime(column driver.ColumnDefinition) string {
	return r.TypeTimestamp(column)
}

func (r *Grammar) TypeDateTimeTz(column driver.ColumnDefinition) string {
	return r.TypeDateTime(column)
}

func (r *Grammar) TypeDecimal(column driver.ColumnDefinition) string {
	return "numeric"
}

func (r *Grammar) TypeDouble(column driver.ColumnDefinition) string {
	return "double"
}

func (r *Grammar) TypeEnum(column driver.ColumnDefinition) string {
	return fmt.Sprintf(`varchar check ("%s" in (%s))`, column.GetName(), strings.Join(r.wrap.Quotes(cast.ToStringSlice(column.GetAllowed())), ", "))
}

func (r *Grammar) TypeFloat(column driver.ColumnDefinition) string {
	return "float"
}

func (r *Grammar) TypeInteger(column driver.ColumnDefinition) string {
	return "integer"
}

func (r *Grammar) TypeJson(column driver.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeJsonb(column driver.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeLongText(column driver.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeMediumInteger(column driver.ColumnDefinition) string {
	return "integer"
}

func (r *Grammar) TypeMediumText(column driver.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeSmallInteger(column driver.ColumnDefinition) string {
	return "integer"
}

func (r *Grammar) TypeString(column driver.ColumnDefinition) string {
	return "varchar"
}

func (r *Grammar) TypeText(column driver.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeTime(column driver.ColumnDefinition) string {
	return "time"
}

func (r *Grammar) TypeTimeTz(column driver.ColumnDefinition) string {
	return r.TypeTime(column)
}

func (r *Grammar) TypeTimestamp(column driver.ColumnDefinition) string {
	if column.GetUseCurrent() {
		column.Default(schema.Expression("CURRENT_TIMESTAMP"))
	}

	return "datetime"
}

func (r *Grammar) TypeTimestampTz(column driver.ColumnDefinition) string {
	return r.TypeTimestamp(column)
}

func (r *Grammar) TypeTinyInteger(column driver.ColumnDefinition) string {
	return "integer"
}

func (r *Grammar) TypeTinyText(column driver.ColumnDefinition) string {
	return "text"
}

func (r *Grammar) TypeUuid(column driver.ColumnDefinition) string {
	return "varchar"
}

func (r *Grammar) addForeignKeys(commands []*driver.Command) string {
	var sql string

	for _, command := range commands {
		sql += r.getForeignKey(command)
	}

	return sql
}

func (r *Grammar) addPrimaryKeys(command *driver.Command) string {
	if command == nil {
		return ""
	}

	return fmt.Sprintf(", primary key (%s)", r.wrap.Columnize(command.Columns))
}

func (r *Grammar) getColumns(blueprint driver.Blueprint) []string {
	var columns []string
	for _, column := range blueprint.GetAddedColumns() {
		columns = append(columns, r.getColumn(blueprint, column))
	}

	return columns
}

func (r *Grammar) getColumn(blueprint driver.Blueprint, column driver.ColumnDefinition) string {
	sql := fmt.Sprintf("%s %s", r.wrap.Column(column.GetName()), schema.ColumnType(r, column))

	for _, modifier := range r.modifiers {
		sql += modifier(blueprint, column)
	}

	return sql
}

func (r *Grammar) getForeignKey(command *driver.Command) string {
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

func getCommandByName(commands []*driver.Command, name string) *driver.Command {
	commands = getCommandsByName(commands, name)
	if len(commands) == 0 {
		return nil
	}

	return commands[0]
}

func getCommandsByName(commands []*driver.Command, name string) []*driver.Command {
	var filteredCommands []*driver.Command
	for _, command := range commands {
		if command.Name == name {
			filteredCommands = append(filteredCommands, command)
		}
	}

	return filteredCommands
}
