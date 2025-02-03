package sqlite

import "gorm.io/gorm/clause"

type Query struct {
}

func NewQuery() *Query {
	return &Query{}
}

func (r *Query) LockForUpdate() clause.Expression {
	return nil
}

func (r *Query) RandomOrder() string {
	return "RANDOM()"
}

func (r *Query) SharedLock() clause.Expression {
	return nil
}
