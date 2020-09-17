// Package rule ...
// Maintainer : LibertusDio
// DO NOT EDIT directly
package rule

import (
	"encoding/json"

	uuid "github.com/satori/go.uuid"
)

type Condition struct {
	Type      int    `json:"type"`
	LeftSide  string `json:"left_side"`
	LeftType  int    `json:"left_type"`
	Compare   int    `json:"compare"`
	RightSide string `json:"right_side"`
	RightType int    `json:"right_type"`
}

type JSONmap struct {
	Name  string `json:"name"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Modifer struct {
	Sequence    int    `json:"sequence"`
	Operand     int    `json:"operand"`
	DataType    int    `json:"data_type"`
	LeftSide    string `json:"left_side"`
	LeftType    int    `json:"left_type"`
	RightSide   string `json:"right_side"`
	RightType   int    `json:"right_type"`
	TargetField string `json:"target_field"`
}

type ModiferComplex struct {
	Flat      int64     `json:"flat"`
	Percent   int64     `json:"percentage"`
	Selection []JSONmap `json:"select"`
}

func (rm *ModiferComplex) Select(key string) *JSONmap {
	for _, item := range rm.Selection {
		if item.Key == key {
			return &item
		}
	}
	return nil
}

type Rule struct {
	ConditionChain []Condition `json:"condition_chain"`
	ModiferChain   []Modifer   `json:"rate_modifer"`
}

type RuleSetting struct {
	ID          string `json:"id" gorm:"primary_key"`
	Enable      bool   `json:"enable"`
	BreakOnFail bool   `json:"break_on_fail"`
	Sequence    int64  `json:"sequence"`
	RuleID      string `json:"rule_id"`
	RuleType    int    `json:"rule_type"` // for filtering
	Rule        Rule   `json:"rule"`
}

type RuleSettingDB struct {
	ID          string `json:"id" gorm:"primary_key"`
	Enable      bool   `json:"enable"`
	BreakOnFail bool   `json:"break_on_fail"`
	Sequence    int64  `json:"sequence"`
	RuleID      string `json:"rule_id"`
	RuleType    int    `json:"rule_type"`
	Rule        string `json:"rule"`
}

func (rs *RuleSetting) MakeDBObject() (*RuleSettingDB, error) {
	rr, err := json.Marshal(rs.Rule)
	if err != nil {
		return nil, err
	}
	if rs.ID == "" {
		rs.ID = uuid.NewV4().String()
	}
	rsdb := &RuleSettingDB{
		ID:          rs.ID,
		Enable:      rs.Enable,
		BreakOnFail: rs.BreakOnFail,
		Sequence:    rs.Sequence,
		RuleID:      rs.RuleID,
		RuleType:    rs.RuleType,
		Rule:        string(rr),
	}

	return rsdb, nil
}

func (rs *RuleSettingDB) MakeObject() (*RuleSetting, error) {
	rr := new(Rule)
	err := json.Unmarshal([]byte(rs.Rule), rr)
	if err != nil {
		return nil, err
	}
	rsdb := &RuleSetting{
		ID:          rs.ID,
		Enable:      rs.Enable,
		BreakOnFail: rs.BreakOnFail,
		Sequence:    rs.Sequence,
		RuleID:      rs.RuleID,
		RuleType:    rs.RuleType,
		Rule:        *rr,
	}

	return rsdb, nil
}
