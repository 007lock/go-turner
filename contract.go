// Package rule ...
// Maintainer : LibertusDio
// DO NOT EDIT directly
package rule

// Engine the engine that crunch the rules
type Engine interface {
	ApplySetting(rqr interface{}, rs RuleSetting) (bool, bool, error)
	ApplySettings(rqr interface{}, rs []RuleSetting) (bool, error)
	ApplyModifer(rqr interface{}, rm Modifer) (bool, error)
	CheckRuleCondition(rqr interface{}, c Condition) (bool, error)
}

// Supply to fetch and save rules
type Supply interface {
	SaveRuleSettings(rs []RuleSetting) error
	FetchRuleSettings(id string, start int) ([]RuleSetting, error)
}
