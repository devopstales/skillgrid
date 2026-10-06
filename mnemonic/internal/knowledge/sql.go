package knowledge

import (
	"path/filepath"
	"regexp"
	"strings"
)

// SQLTable is one table parsed from SQL DDL: its name and its columns.
type SQLTable struct {
	Name    string
	Columns []string
}

// SQLAccess is one read/write of a table from SQL DML: the table name, the
// operation (reads | writes), and the line it appears on.
type SQLAccess struct {
	Table    string
	Op       string // KindReads | KindWrites
	Line     int
	Confidence string
}

// SQLResult is the extraction result for one SQL file: the tables (DDL) it
// defines and the read/writes (DML) it performs.
type SQLResult struct {
	Path    string
	Tables  []SQLTable
	Access  []SQLAccess
	IsSQL   bool
}

// sqlTable matches a CREATE TABLE statement header: the table name.
var sqlTable = regexp.MustCompile("(?i)\\bCREATE\\s+TABLE\\s+(?:IF\\s+NOT\\s+EXISTS\\s+)?[`\"'\\[]?([A-Za-z_][A-Za-z0-9_]*)[`\"'\\]]?")

// sqlColumn matches a column definition inside a CREATE TABLE body: the
// column name (the first token of a body line, when it is an identifier not
// followed by a constraint keyword).
var sqlColumn = regexp.MustCompile("(?i)^\\s*([A-Za-z_][A-Za-z0-9_]*)\\s+(?:INTEGER|INT|TEXT|VARCHAR|CHAR|BOOLEAN|BOOL|REAL|DOUBLE|FLOAT|DECIMAL|NUMERIC|BLOB|DATE|DATETIME|TIMESTAMP|TIMESTAMPTZ|UUID|JSON|JSONB|BIGINT|SMALLINT|MEDIUMINT|SERIAL|BIGSERIAL)\\b")

// sqlSelect matches a SELECT statement naming its target table(s).
var sqlSelect = regexp.MustCompile("(?i)\\bFROM\\s+[`\"'\\[]?([A-Za-z_][A-Za-z0-9_]*)[`\"'\\]]?")

// sqlInsertInto matches an INSERT INTO statement naming its target table.
var sqlInsertInto = regexp.MustCompile("(?i)\\bINSERT\\s+INTO\\s+[`\"'\\[]?([A-Za-z_][A-Za-z0-9_]*)[`\"'\\]]?")

// sqlUpdate matches an UPDATE statement naming its target table.
var sqlUpdate = regexp.MustCompile("(?i)\\bUPDATE\\s+[`\"'\\[]?([A-Za-z_][A-Za-z0-9_]*)[`\"'\\]]?")

// sqlDeleteFrom matches a DELETE FROM statement naming its target table.
var sqlDeleteFrom = regexp.MustCompile("(?i)\\bDELETE\\s+FROM\\s+[`\"'\\[]?([A-Za-z_][A-Za-z0-9_]*)[`\"'\\]]?")

// sqlStatementSplits splits a SQL script into individual statements at
// semicolons (respecting single/double quotes so a `;` inside a string literal
// does not split). An unbalanced-paren statement swallows subsequent text up
// to the next `;`, so the split is conservative: each chunk is then validated
// by isBalanced (a malformed statement is skipped, per 03.7).
func sqlStatementSplits(src []byte) []string {
	var stmts []string
	inSQuote, inDQuote := false, false
	var cur strings.Builder
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case inSQuote:
			cur.WriteByte(c)
			if c == '\'' {
				inSQuote = false
			}
		case inDQuote:
			cur.WriteByte(c)
			if c == '"' {
				inDQuote = false
			}
		case c == '\'':
			inSQuote = true
			cur.WriteByte(c)
		case c == '"':
			inDQuote = true
			cur.WriteByte(c)
		case c == ';':
			stmts = append(stmts, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if strings.TrimSpace(cur.String()) != "" {
		stmts = append(stmts, cur.String())
	}
	return stmts
}

// isBalanced reports whether a statement has balanced parens (a malformed
// statement with unbalanced parens is skipped, per 03.7).
func isBalanced(s string) bool {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

// startsWithCREATE reports whether a statement is a DDL statement (CREATE /
// ALTER / DROP).
func startsWithCREATE(stmt string) bool {
	upper := strings.ToUpper(strings.TrimSpace(stmt))
	return strings.HasPrefix(upper, "CREATE") || strings.HasPrefix(upper, "ALTER") || strings.HasPrefix(upper, "DROP")
}

// containsDML reports whether a statement (possibly a swallowed fragment)
// contains a DML keyword (SELECT / INSERT / UPDATE / DELETE) — used to recover
// a valid DML that was swallowed by an unbalanced-paren DDL fragment.
func containsDML(stmt string) bool {
	upper := strings.ToUpper(stmt)
	for _, kw := range []string{"SELECT", "INSERT", "UPDATE", "DELETE"} {
		if strings.Contains(upper, kw) {
			return true
		}
	}
	return false
}

// leadingDDLName extracts the leading DDL statement's table name (for a
// swallowed fragment like `CREATE TABLE broken (... INSERT INTO good1 ...`),
// so the malformed DDL's table can be excluded from the indexed nodes.
func leadingDDLName(stmt string) string {
	if m := sqlTable.FindStringSubmatch(stmt); m != nil {
		return m[1]
	}
	return ""
}

// firstTableLine returns the line number (1-based) of the first non-blank,
// non-comment character of a statement within the source.
func firstTableLine(src []byte, stmt string) int {
	s := strings.TrimSpace(stmt)
	if s == "" {
		return 0
	}
	idx := strings.Index(string(src), s)
	if idx < 0 {
		return 0
	}
	return strings.Count(string(src[:idx]), "\n") + 1
}

// ExtractSQL parses one SQL file into its tables (DDL) and read/writes (DML).
// It never errors and never returns nil: a malformed statement (unbalanced
// parens) is skipped and the rest is indexed (03.7). A non-SQL path returns an
// empty (IsSQL=false) result.
func ExtractSQL(path string, src []byte) *SQLResult {
	res := &SQLResult{Path: path, IsSQL: true}
	for _, raw := range sqlStatementSplits(src) {
		stmt := strings.TrimSpace(raw)
		if stmt == "" {
			continue
		}
		line := firstTableLine(src, raw)
		// A malformed DDL statement (unbalanced parens, no DML) is skipped
		// (03.7). A swallowed fragment that contains a DML is still processed
		// for its own table reference (the index does not abort on the
		// malformed DDL); the malformed DDL's leading table is excluded.
		if startsWithCREATE(stmt) && !isBalanced(stmt) {
			if !containsDML(stmt) {
				continue
			}
			res.processStatement(stmt, line, leadingDDLName(stmt))
			continue
		}
		res.processStatement(stmt, line, "")
	}
	// De-dup tables (a table defined once) and accesses.
	res.dedup()
	return res
}

// isSQLPath reports whether path is a .sql file (the primary DDL source).
func isSQLPath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".sql")
}

// processStatement parses one SQL statement into tables (DDL) or read/writes
// (DML). excludeTable, when non-empty, suppresses a table node for that name
// (a malformed DDL's leading table in a swallowed fragment).
func (r *SQLResult) processStatement(stmt string, line int, excludeTable string) {
	if m := sqlTable.FindStringSubmatch(stmt); m != nil {
		name := m[1]
		if name != excludeTable {
			table := SQLTable{Name: name}
			// Parse the column body (between the first '(' after the table
			// name and its matching ')').
			body := columnBody(stmt, name)
			for _, colLine := range strings.Split(body, ",") {
				if cm := sqlColumn.FindStringSubmatch(colLine); cm != nil {
					table.Columns = append(table.Columns, cm[1])
				}
			}
			r.Tables = append(r.Tables, table)
			return
		}
		// The malformed DDL's leading table is excluded; fall through to its
		// DML (the swallowed SELECT / INSERT / UPDATE / DELETE).
	}
	// DML: reads (SELECT) and writes (INSERT/UPDATE/DELETE).
	if m := sqlSelect.FindStringSubmatch(stmt); m != nil && selectIsRead(stmt) {
		r.Access = append(r.Access, SQLAccess{Table: m[1], Op: KindReads, Line: line, Confidence: ConfidenceExtracted})
	}
	if m := sqlInsertInto.FindStringSubmatch(stmt); m != nil {
		r.Access = append(r.Access, SQLAccess{Table: m[1], Op: KindWrites, Line: line, Confidence: ConfidenceExtracted})
	}
	if m := sqlUpdate.FindStringSubmatch(stmt); m != nil {
		r.Access = append(r.Access, SQLAccess{Table: m[1], Op: KindWrites, Line: line, Confidence: ConfidenceExtracted})
	}
	if m := sqlDeleteFrom.FindStringSubmatch(stmt); m != nil {
		r.Access = append(r.Access, SQLAccess{Table: m[1], Op: KindWrites, Line: line, Confidence: ConfidenceExtracted})
	}
}

// selectIsRead reports whether a statement contains a SELECT (a read). Contains
// subsumes the prefix case, so a top-level or embedded SELECT both count.
func selectIsRead(stmt string) bool {
	upper := strings.ToUpper(strings.TrimSpace(stmt))
	return strings.Contains(upper, "SELECT")
}

// columnBody extracts the column-definition body of a CREATE TABLE statement
// (the text between the first '(' after the table name and its matching
// ')').
func columnBody(stmt, name string) string {
	idx := strings.Index(stmt, name)
	if idx < 0 {
		return ""
	}
	rest := stmt[idx+len(name):]
	open := strings.Index(rest, "(")
	if open < 0 {
		return ""
	}
	depth := 0
	for i := open; i < len(rest); i++ {
		switch rest[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return rest[open+1 : i]
			}
		}
	}
	return rest[open+1:]
}

// dedup de-duplicates tables (by name, keeping the first) and accesses (by
// table+op, keeping the first) so a re-index does not double-count.
func (r *SQLResult) dedup() {
	seenTable := map[string]bool{}
	tabs := r.Tables[:0]
	for _, t := range r.Tables {
		if seenTable[t.Name] {
			continue
		}
		seenTable[t.Name] = true
		tabs = append(tabs, t)
	}
	r.Tables = tabs
	seenAccess := map[string]bool{}
	acc := r.Access[:0]
	for _, a := range r.Access {
		k := a.Table + "|" + a.Op
		if seenAccess[k] {
			continue
		}
		seenAccess[k] = true
		acc = append(acc, a)
	}
	r.Access = acc
}
