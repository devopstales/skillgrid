package config_test

import "github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"

// storeDialectSQLite isolates the store import to one spot so the config
// tests read as pure-value assertions.
func storeDialectSQLite() store.Dialect { return store.SQLite }
