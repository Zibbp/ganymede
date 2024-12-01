// Code generated by ent, DO NOT EDIT.

package sessions

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/zibbp/ganymede/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id int) predicate.Sessions {
	return predicate.Sessions(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id int) predicate.Sessions {
	return predicate.Sessions(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id int) predicate.Sessions {
	return predicate.Sessions(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...int) predicate.Sessions {
	return predicate.Sessions(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...int) predicate.Sessions {
	return predicate.Sessions(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id int) predicate.Sessions {
	return predicate.Sessions(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id int) predicate.Sessions {
	return predicate.Sessions(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id int) predicate.Sessions {
	return predicate.Sessions(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id int) predicate.Sessions {
	return predicate.Sessions(sql.FieldLTE(FieldID, id))
}

// Token applies equality check predicate on the "token" field. It's identical to TokenEQ.
func Token(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldEQ(FieldToken, v))
}

// Data applies equality check predicate on the "data" field. It's identical to DataEQ.
func Data(v []byte) predicate.Sessions {
	return predicate.Sessions(sql.FieldEQ(FieldData, v))
}

// Expiry applies equality check predicate on the "expiry" field. It's identical to ExpiryEQ.
func Expiry(v time.Time) predicate.Sessions {
	return predicate.Sessions(sql.FieldEQ(FieldExpiry, v))
}

// TokenEQ applies the EQ predicate on the "token" field.
func TokenEQ(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldEQ(FieldToken, v))
}

// TokenNEQ applies the NEQ predicate on the "token" field.
func TokenNEQ(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldNEQ(FieldToken, v))
}

// TokenIn applies the In predicate on the "token" field.
func TokenIn(vs ...string) predicate.Sessions {
	return predicate.Sessions(sql.FieldIn(FieldToken, vs...))
}

// TokenNotIn applies the NotIn predicate on the "token" field.
func TokenNotIn(vs ...string) predicate.Sessions {
	return predicate.Sessions(sql.FieldNotIn(FieldToken, vs...))
}

// TokenGT applies the GT predicate on the "token" field.
func TokenGT(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldGT(FieldToken, v))
}

// TokenGTE applies the GTE predicate on the "token" field.
func TokenGTE(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldGTE(FieldToken, v))
}

// TokenLT applies the LT predicate on the "token" field.
func TokenLT(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldLT(FieldToken, v))
}

// TokenLTE applies the LTE predicate on the "token" field.
func TokenLTE(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldLTE(FieldToken, v))
}

// TokenContains applies the Contains predicate on the "token" field.
func TokenContains(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldContains(FieldToken, v))
}

// TokenHasPrefix applies the HasPrefix predicate on the "token" field.
func TokenHasPrefix(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldHasPrefix(FieldToken, v))
}

// TokenHasSuffix applies the HasSuffix predicate on the "token" field.
func TokenHasSuffix(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldHasSuffix(FieldToken, v))
}

// TokenEqualFold applies the EqualFold predicate on the "token" field.
func TokenEqualFold(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldEqualFold(FieldToken, v))
}

// TokenContainsFold applies the ContainsFold predicate on the "token" field.
func TokenContainsFold(v string) predicate.Sessions {
	return predicate.Sessions(sql.FieldContainsFold(FieldToken, v))
}

// DataEQ applies the EQ predicate on the "data" field.
func DataEQ(v []byte) predicate.Sessions {
	return predicate.Sessions(sql.FieldEQ(FieldData, v))
}

// DataNEQ applies the NEQ predicate on the "data" field.
func DataNEQ(v []byte) predicate.Sessions {
	return predicate.Sessions(sql.FieldNEQ(FieldData, v))
}

// DataIn applies the In predicate on the "data" field.
func DataIn(vs ...[]byte) predicate.Sessions {
	return predicate.Sessions(sql.FieldIn(FieldData, vs...))
}

// DataNotIn applies the NotIn predicate on the "data" field.
func DataNotIn(vs ...[]byte) predicate.Sessions {
	return predicate.Sessions(sql.FieldNotIn(FieldData, vs...))
}

// DataGT applies the GT predicate on the "data" field.
func DataGT(v []byte) predicate.Sessions {
	return predicate.Sessions(sql.FieldGT(FieldData, v))
}

// DataGTE applies the GTE predicate on the "data" field.
func DataGTE(v []byte) predicate.Sessions {
	return predicate.Sessions(sql.FieldGTE(FieldData, v))
}

// DataLT applies the LT predicate on the "data" field.
func DataLT(v []byte) predicate.Sessions {
	return predicate.Sessions(sql.FieldLT(FieldData, v))
}

// DataLTE applies the LTE predicate on the "data" field.
func DataLTE(v []byte) predicate.Sessions {
	return predicate.Sessions(sql.FieldLTE(FieldData, v))
}

// ExpiryEQ applies the EQ predicate on the "expiry" field.
func ExpiryEQ(v time.Time) predicate.Sessions {
	return predicate.Sessions(sql.FieldEQ(FieldExpiry, v))
}

// ExpiryNEQ applies the NEQ predicate on the "expiry" field.
func ExpiryNEQ(v time.Time) predicate.Sessions {
	return predicate.Sessions(sql.FieldNEQ(FieldExpiry, v))
}

// ExpiryIn applies the In predicate on the "expiry" field.
func ExpiryIn(vs ...time.Time) predicate.Sessions {
	return predicate.Sessions(sql.FieldIn(FieldExpiry, vs...))
}

// ExpiryNotIn applies the NotIn predicate on the "expiry" field.
func ExpiryNotIn(vs ...time.Time) predicate.Sessions {
	return predicate.Sessions(sql.FieldNotIn(FieldExpiry, vs...))
}

// ExpiryGT applies the GT predicate on the "expiry" field.
func ExpiryGT(v time.Time) predicate.Sessions {
	return predicate.Sessions(sql.FieldGT(FieldExpiry, v))
}

// ExpiryGTE applies the GTE predicate on the "expiry" field.
func ExpiryGTE(v time.Time) predicate.Sessions {
	return predicate.Sessions(sql.FieldGTE(FieldExpiry, v))
}

// ExpiryLT applies the LT predicate on the "expiry" field.
func ExpiryLT(v time.Time) predicate.Sessions {
	return predicate.Sessions(sql.FieldLT(FieldExpiry, v))
}

// ExpiryLTE applies the LTE predicate on the "expiry" field.
func ExpiryLTE(v time.Time) predicate.Sessions {
	return predicate.Sessions(sql.FieldLTE(FieldExpiry, v))
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Sessions) predicate.Sessions {
	return predicate.Sessions(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Sessions) predicate.Sessions {
	return predicate.Sessions(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Sessions) predicate.Sessions {
	return predicate.Sessions(sql.NotPredicates(p))
}