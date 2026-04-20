package model

import "github.com/QuantumNous/new-api/common"

// GetDBTimestamp returns a UNIX timestamp from database time.
// Falls back to application time on error.
func GetDBTimestamp() int64 {
	var ts int64
	var err error
	switch {
	case common.UsingPostgreSQL:
		err = DB.Raw("SELECT EXTRACT(EPOCH FROM NOW())::bigint").Scan(&ts).Error
	case common.UsingSQLite:
		// SQLite tests commonly run with a single in-memory connection.
		// Querying database time from inside an open transaction can self-block,
		// while SQLite and application time are effectively the same process clock.
		return common.GetTimestamp()
	default:
		err = DB.Raw("SELECT UNIX_TIMESTAMP()").Scan(&ts).Error
	}
	if err != nil || ts <= 0 {
		return common.GetTimestamp()
	}
	return ts
}
