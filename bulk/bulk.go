// Copyright © 2019 Martin Tournoij <martin@arp242.net>
// This file is part of GoatCounter and published under the terms of the AGPLv3,
// which can be found in the LICENSE file or at gnu.org/licenses/agpl.html

// Package bulk provides helpers for bulk SQL operations.
package bulk

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// Insert as many rows as possible per query we send to the server.
type Insert struct {
	rows    uint16
	limit   uint16
	ctx     context.Context
	db      *sqlx.DB
	table   string
	columns []string
	insert  squirrel.InsertBuilder
	errors  []error
}

// NewInsert makes a new Insert builder.
func NewInsert(ctx context.Context, db *sqlx.DB, table string, columns []string) Insert {
	return Insert{
		ctx: ctx,
		db:  db,
		// SQLITE_MAX_VARIABLE_NUMBER: https://www.sqlite.org/limits.html
		limit:   uint16(999/len(columns) - 1),
		table:   table,
		columns: columns,
		insert: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).
			Insert(table).Columns(columns...),
	}
}

// Values adds a set of values.
func (m *Insert) Values(values ...interface{}) {
	m.insert = m.insert.Values(values...)
	m.rows++

	if m.rows >= m.limit {
		m.doInsert()
	}
}

// Finish the operation, returning any errors.
func (m *Insert) Finish() error {
	if m.rows > 0 {
		m.doInsert()
	}

	if len(m.errors) == 0 {
		return nil
	}

	return fmt.Errorf("%d errors: %#v", len(m.errors), m.errors)
}

func (m *Insert) doInsert() {
	query, args, err := m.insert.ToSql()
	if err != nil {
		m.errors = append(m.errors, err)
		goto reset
	}

	_, err = m.db.ExecContext(m.ctx, query, args...)
	if err != nil {
		m.errors = append(m.errors, err)
		goto reset
	}

reset:
	m.insert = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).
		Insert(m.table).Columns(m.columns...)
	m.rows = 0
}
