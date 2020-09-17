// Package rule ...
// Maintainer : LibertusDio
// DO NOT EDIT directly
package rule

import (
	"errors"
	"time"
)

const DB_TABLE_RULE string = "rule_settings"
const DB_TABLE_INFO string = "rule_infos"

type ruleconditioncompare struct {
	EQUAL      int
	MORE       int
	LESS       int
	MORE_EQUAL int
	LESS_EQUAL int
	IN         int
	NOT        int
	PACKAGE    int
	NOT_IN     int
}

var RuleConditionCompare = ruleconditioncompare{
	EQUAL:      0,
	MORE:       1,
	LESS:       2,
	MORE_EQUAL: 3,
	LESS_EQUAL: 4,
	IN:         5,
	NOT:        6,
	PACKAGE:    7,
	NOT_IN:     8,
}

type ruleconditiontype struct {
	DAY_OF_WEEK int
	DATE        int
	STRING      int
	INT         int
	MUST        int
}

var RuleConditionType = ruleconditiontype{
	DAY_OF_WEEK: 0,
	DATE:        1,
	STRING:      2,
	INT:         3,
	MUST:        4,
}

var DayOfWeek = map[string]time.Time{
	"Sunday":    time.Unix(1567339200, 0),
	"Monday":    time.Unix(1567425600, 0),
	"Tuesday":   time.Unix(1567512000, 0),
	"Wednesday": time.Unix(1567598400, 0),
	"Thursday":  time.Unix(1567684800, 0),
	"Friday":    time.Unix(1567771200, 0),
	"Saturday":  time.Unix(1567857600, 0),
}

var DayOfWeekUnix = map[string]string{
	"Sunday":    "1567339200",
	"Monday":    "1567425600",
	"Tuesday":   "1567512000",
	"Wednesday": "1567598400",
	"Thursday":  "1567684800",
	"Friday":    "1567771200",
	"Saturday":  "1567857600",
}

type ruleoperand struct {
	SET int
	ADD int
	SUB int
	MLT int
	DIV int
	SEL int
	SUM int
}

var RuleOperand = ruleoperand{
	SET: 0,
	ADD: 1,
	SUB: 2,
	MLT: 3,
	DIV: 4,
	SEL: 5,
	SUM: 6,
}

type ruleselectoperand struct {
	SET int
	ADD int
	SUB int
	MLT int
	DIV int
}

var RuleSelectOperand = ruleselectoperand{
	SET: 0,
	ADD: 1,
	SUB: 2,
	MLT: 3,
	DIV: 4,
}

type conditionsidetype struct {
	FIELD int
	VALUE int
}

var ConditionSideType = conditionsidetype{
	FIELD: 102,
	VALUE: 118,
}

type modifersidetype struct {
	FIELD   int
	VALUE   int
	COMPLEX int
}

var ModiferSideType = modifersidetype{
	FIELD:   102,
	VALUE:   118,
	COMPLEX: 99,
}

type modiferdatatype struct {
	STRING int
	INT    int
	JMP    int
	JRT    int
}

var ModiferDataType = modiferdatatype{
	STRING: 0,
	INT:    1,
	JMP:    90,
	JRT:    91,
}

type rulesettingerror struct {
	CONDITION_SIDE_INVALID    error
	UNSUPPORTED_OPERATION     error
	SETTING_NOT_IN_ORDER      error
	MODIFER_SIDE_INVALID      error
	MODIFER_FEILD_NOT_EXISTED error
	DIV_BY_ZERO               error
	UNABLE_TO_FETCH           error
}

var RuleSettingError = rulesettingerror{
	CONDITION_SIDE_INVALID:    errors.New("Invalid condition side"),
	MODIFER_SIDE_INVALID:      errors.New("Invalid modifer side"),
	MODIFER_FEILD_NOT_EXISTED: errors.New("Field not existed"),
	UNSUPPORTED_OPERATION:     errors.New("Unsupported operation"),
	SETTING_NOT_IN_ORDER:      errors.New("Rule Settings not in order"),
	DIV_BY_ZERO:               errors.New("Divide by zero"),
	UNABLE_TO_FETCH:           errors.New("Unable to fetch next rule set."),
}

type rulesettingstep struct {
	BASE      int
	ROOM_TYPE int
	SEASON    int
	OCCUPANCY int
	PROMO     int
	DC        int
	PACKAGE   int
}

var RuleSettingStep = rulesettingstep{
	BASE:      0,
	ROOM_TYPE: 1,
	SEASON:    2,
	OCCUPANCY: 3,
	PACKAGE:   4,
	DC:        5,
	PROMO:     6,
}
