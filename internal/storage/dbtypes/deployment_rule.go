package dbtypes

type DeploymentRuleRow struct {
	ID               int64   `db:"id"`
	ProfileID        int64   `db:"profile_id"`
	GameRelativePath string  `db:"game_relative_path"`
	RuleKind         string  `db:"rule_kind"`
	WinnerModID      *int64  `db:"winner_mod_id"`
	Explanation      *string `db:"explanation"`
	CreatedAt        string  `db:"created_at"`
}
