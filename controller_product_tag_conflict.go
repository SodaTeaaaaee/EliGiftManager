package main

import "gorm.io/gorm/clause"

// SQLite requires the ON CONFLICT target predicate for a partial unique index
// to match the index WHERE clause literally. Parameterized predicates generated
// from clause.Eq do not match `WHERE tag_type = 'identity'/'user'`.
func identityTagConflictTargetWhere() clause.Where {
	return clause.Where{
		Exprs: []clause.Expression{
			clause.Expr{SQL: "tag_type = 'identity'"},
		},
	}
}

func userTagConflictTargetWhere() clause.Where {
	return clause.Where{
		Exprs: []clause.Expression{
			clause.Expr{SQL: "tag_type = 'user'"},
		},
	}
}
