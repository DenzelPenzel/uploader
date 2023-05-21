package wrapper

import "database/sql"

type SqlDB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}
