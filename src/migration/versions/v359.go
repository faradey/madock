package versions

// V359 re-runs db/type migration for users who upgraded past v3.4.0
// before the migration existed. The migration is idempotent — it skips
// projects that already have db/type set.
func V359() {
	V340()
}
