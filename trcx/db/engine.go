package db

import (
	eUtils "tierceron/utils"

	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/sql"
)

type TierceronTable struct {
	Table  *memory.Table
	Schema sql.Schema
}

type TierceronEngine struct {
	Config     eUtils.DriverConfig
	Database   *memory.Database
	Engine     *sqle.Engine
	Context    *sql.Context
	TableCache map[string]*TierceronTable
}
