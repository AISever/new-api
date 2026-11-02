// Isolated restore verification only. This executable refuses production DSNs.
package main

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func main() {
	dsn, err := url.Parse(os.Getenv("SQL_DSN"))
	if err != nil || dsn.Hostname() != "new-api-release-restore-20261102-r2-postgres" || dsn.Path != "/new-api" {
		panic("migration rehearsal requires the dedicated isolated restore database")
	}
	common.IsMasterNode = true
	start := time.Now()
	if err := model.InitDB(); err != nil {
		panic(err)
	}
	db, err := model.DB.DB()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	fmt.Printf("isolated migration succeeded in %s\n", time.Since(start))
}
