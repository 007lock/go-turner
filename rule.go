// Package rule ...
// Maintainer : LibertusDio
// DO NOT EDIT directly
package rule

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CloudHMS/hms.loyalty.core/pkg/util"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

type ruleEngine struct {
	sp Supply
}

func NewEngine(sp Supply) Engine {
	return &ruleEngine{sp}
}

// ApplySetting Check conditions and apply settings from for single rule
func (re *ruleEngine) ApplySetting(rqr interface{}, rs RuleSetting) (bool, bool, error) {
	var wg sync.WaitGroup
	wg.Add(len(rs.Rule.ConditionChain))
	var resultSum int
	resultSum = 0
	var grtn error
	for _, condition := range rs.Rule.ConditionChain {
		go func(condition Condition) {
			temp := reflect.Indirect(reflect.ValueOf(rqr)).Interface()
			result, err := re.CheckRuleCondition(temp, condition)
			if err != nil && grtn == nil {
				grtn = err
			}

			if !result {
				resultSum--
			}
			wg.Done()
		}(condition)
	}

	wg.Wait()
	if grtn != nil {
		return false, false, grtn
	}
	if resultSum < 0 {
		if rs.BreakOnFail {
			return false, true, nil
		}
		return false, false, nil
	}

	// Apply modifers
	sort.Slice(rs.Rule.ModiferChain, func(i, j int) bool {
		return rs.Rule.ModiferChain[i].Sequence < rs.Rule.ModiferChain[j].Sequence
	})

	for _, modifer := range rs.Rule.ModiferChain {
		if modifer.DataType == ModiferDataType.JMP {
			// Jump then leave
			id := modifer.LeftSide
			start, err := strconv.ParseInt(modifer.RightSide, 10, 64)
			if err != nil {
				return false, false, RuleSettingError.MODIFER_SIDE_INVALID
			}

			rsn, err := re.sp.FetchRuleSettings(id, int(start))
			if err != nil {
				return false, false, RuleSettingError.UNABLE_TO_FETCH
			}
			rs, er := re.ApplySettings(rqr, rsn)
			return rs, true, er
		}
		result, err := re.ApplyModifer(rqr, modifer)

		if err != nil {
			// TODO: log error
			return false, false, err
		}
		if !result {
			// TODO: log error
			return false, false, nil
		}
	}
	return true, false, nil
}

// ApplySettings Check conditions and apply settings from rule set
func (re *ruleEngine) ApplySettings(rqr interface{}, rs []RuleSetting) (bool, error) {
	// check settings sequence
	cs := re.checkSettingSequence(rs)
	if !cs {
		return false, RuleSettingError.SETTING_NOT_IN_ORDER
	}

	for _, setting := range rs {
		result, br, err := re.ApplySetting(rqr, setting)
		if err != nil {
			return false, err
		}
		if !result {
			// TODO: log
		}
		if br {
			return result, nil
		}
	}
	return true, nil
}

// ApplyModifer apply the Modifer directly to the rqr data
func (re *ruleEngine) ApplyModifer(rqr interface{}, rm Modifer) (bool, error) {
	switch rm.DataType {
	case ModiferDataType.STRING:
		if err := re.applyModiferString(rqr, rm); err != nil {
			return false, re.applyModiferString(rqr, rm)
		}
		return true, nil
	case ModiferDataType.INT:
		if err := re.applyModiferInt(rqr, rm); err != nil {
			return false, re.applyModiferInt(rqr, rm)
		}
		return true, nil
	case ModiferDataType.JRT:
		return re.applyModiferJumpReturn(rqr, rm)
	}

	return false, RuleSettingError.UNSUPPORTED_OPERATION
}

func (re *ruleEngine) applyModiferString(rqr interface{}, rm Modifer) error {
	switch rm.Operand {
	case RuleOperand.SET:
		var vl string
		switch rm.LeftType {
		case ModiferSideType.VALUE:
			vl = rm.LeftSide
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.LeftSide)
			vl = temp.String()
		default:
			return RuleSettingError.MODIFER_SIDE_INVALID
		}
		reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.TargetField).SetString(vl)
		return nil

	case RuleOperand.SEL:
		selectWhat := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.LeftSide).String()
		var vr ModiferComplex
		switch rm.RightType {
		case ModiferSideType.VALUE:
			return RuleSettingError.MODIFER_SIDE_INVALID
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.RightSide).Interface().(ModiferComplex)
			vr = temp
		case ModiferSideType.COMPLEX:
			var temp *ModiferComplex
			err := json.Unmarshal([]byte(rm.RightSide), temp)
			if err != nil {
				return err
			}
			vr = *temp
		default:
			return RuleSettingError.MODIFER_SIDE_INVALID
		}

		if sel := vr.Select(selectWhat); sel != nil {
			modifer := sel.Value
			reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.TargetField).SetString(modifer)
			return nil
		}
		return fmt.Errorf("Field not existed %s", selectWhat)
	}
	return RuleSettingError.UNSUPPORTED_OPERATION
}

func (re *ruleEngine) applyModiferInt(rqr interface{}, rm Modifer) error {
	switch rm.Operand {
	case RuleOperand.SET:
		var vl int64
		switch rm.LeftType {
		case ModiferSideType.VALUE:
			temp, err := strconv.ParseInt(rm.LeftSide, 10, 64)
			if err != nil {
				return fmt.Errorf("Invalid left modifer value side %s", rm.LeftSide)
			}
			vl = temp
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.LeftSide)
			vl = temp.Int()
		default:
			return fmt.Errorf("Invalid left modifer side %s", rm.LeftSide)
		}
		reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.TargetField).SetInt(vl)
		return nil
	case RuleOperand.ADD:
		var vl int64
		switch rm.LeftType {
		case ModiferSideType.VALUE:
			temp, err := strconv.ParseInt(rm.LeftSide, 10, 64)
			if err != nil {
				return fmt.Errorf("Invalid left modifer value side %s", rm.LeftSide)
			}
			vl = temp
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.LeftSide)
			vl = temp.Int()
		default:
			return fmt.Errorf("Invalid left modifer side %s", rm.LeftSide)
		}

		var vr int64
		switch rm.RightType {
		case ModiferSideType.VALUE:
			temp, err := strconv.ParseInt(rm.RightSide, 10, 64)
			if err != nil {
				return fmt.Errorf("Invalid right modifer value side %s", rm.RightSide)
			}
			vr = temp
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.RightSide)
			vr = temp.Int()
		case ModiferSideType.COMPLEX:
			var temp *ModiferComplex
			err := json.Unmarshal([]byte(rm.RightSide), temp)
			if err != nil {
				return err
			}
			vr = util.MinInt64(temp.Flat, (temp.Percent*vl)/100)
		default:
			return fmt.Errorf("Invalid right modifer side %s", rm.RightSide)
		}

		reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.TargetField).SetInt(vl + vr)
		return nil
	case RuleOperand.SUB:
		var vl int64
		switch rm.LeftType {
		case ModiferSideType.VALUE:
			temp, err := strconv.ParseInt(rm.LeftSide, 10, 64)
			if err != nil {
				return fmt.Errorf("Invalid left modifer value side %s", rm.LeftSide)
			}
			vl = temp
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.LeftSide)
			vl = temp.Int()
		default:
			return fmt.Errorf("Invalid left modifer side %s", rm.LeftSide)
		}

		var vr int64
		switch rm.RightType {
		case ModiferSideType.VALUE:
			temp, err := strconv.ParseInt(rm.RightSide, 10, 64)
			if err != nil {
				return fmt.Errorf("Invalid right modifer value side %s", rm.RightSide)
			}
			vr = temp
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.RightSide)
			vr = temp.Int()
		case ModiferSideType.COMPLEX:
			var temp *ModiferComplex
			err := json.Unmarshal([]byte(rm.RightSide), temp)
			if err != nil {
				return err
			}
			vr = util.MinInt64(temp.Flat, (temp.Percent*vl)/100)
		default:
			return fmt.Errorf("Invalid right modifer side %s", rm.RightSide)
		}

		reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.TargetField).SetInt(vl - vr)
		return nil
	case RuleOperand.MLT:
		var vl int64
		switch rm.LeftType {
		case ModiferSideType.VALUE:
			temp, err := strconv.ParseInt(rm.LeftSide, 10, 64)
			if err != nil {
				return fmt.Errorf("Invalid left modifer value side %s", rm.LeftSide)
			}
			vl = temp
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.LeftSide)
			vl = temp.Int()
		default:
			return fmt.Errorf("Invalid left modifer side %s", rm.LeftSide)
		}

		var vr int64
		switch rm.RightType {
		case ModiferSideType.VALUE:
			temp, err := strconv.ParseInt(rm.RightSide, 10, 64)
			if err != nil {
				return fmt.Errorf("Invalid right modifer value side %s", rm.RightSide)
			}
			vr = temp
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.RightSide)
			vr = temp.Int()
		case ModiferSideType.COMPLEX:
			var temp *ModiferComplex
			err := json.Unmarshal([]byte(rm.RightSide), temp)
			if err != nil {
				return err
			}
			vr = (temp.Percent * vl) / 100
		default:
			return fmt.Errorf("Invalid right modifer side %s", rm.RightSide)
		}

		reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.TargetField).SetInt(vl * vr)
		return nil
	case RuleOperand.DIV:
		var vl int64
		switch rm.LeftType {
		case ModiferSideType.VALUE:
			temp, err := strconv.ParseInt(rm.LeftSide, 10, 64)
			if err != nil {
				return RuleSettingError.MODIFER_SIDE_INVALID
			}
			vl = temp
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.LeftSide)
			vl = temp.Int()
		default:
			return RuleSettingError.MODIFER_SIDE_INVALID
		}

		var vr int64
		switch rm.RightType {
		case ModiferSideType.VALUE:
			temp, err := strconv.ParseInt(rm.RightSide, 10, 64)
			if err != nil {
				return RuleSettingError.MODIFER_SIDE_INVALID
			}
			vr = temp
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.RightSide)
			vr = temp.Int()
		case ModiferSideType.COMPLEX:
			var temp *ModiferComplex
			err := json.Unmarshal([]byte(rm.RightSide), temp)
			if err != nil {
				return err
			}
			vr = (temp.Percent * vl) / 100
		default:
			return RuleSettingError.MODIFER_SIDE_INVALID
		}
		if vr == 0 {
			return RuleSettingError.DIV_BY_ZERO
		}

		reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.TargetField).SetInt(vl / vr)
		return nil
	case RuleOperand.SEL:
		selectWhat := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.LeftSide).String()
		var vr ModiferComplex
		switch rm.RightType {
		case ModiferSideType.VALUE:
			return RuleSettingError.MODIFER_SIDE_INVALID
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.RightSide).Interface().(ModiferComplex)
			vr = temp
		case ModiferSideType.COMPLEX:
			var temp *ModiferComplex
			err := json.Unmarshal([]byte(rm.RightSide), temp)
			if err != nil {
				return err
			}
			vr = *temp
		default:
			return RuleSettingError.MODIFER_SIDE_INVALID
		}

		if sel := vr.Select(selectWhat); sel != nil {
			modifer, _ := strconv.ParseInt(sel.Value, 10, 64)
			reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.TargetField).SetInt(modifer)
			return nil
		}
		return fmt.Errorf("Field not existed %s", selectWhat)
	case RuleOperand.SUM:
		selectWhat := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.LeftSide).Interface().([]string)
		modifer := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.TargetField).Int()
		var vr ModiferComplex
		switch rm.RightType {
		case ModiferSideType.VALUE:
			return RuleSettingError.MODIFER_SIDE_INVALID
		case ModiferSideType.FIELD:
			temp := reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.RightSide).Interface().(ModiferComplex)
			vr = temp
		case ModiferSideType.COMPLEX:
			var temp *ModiferComplex
			err := json.Unmarshal([]byte(rm.RightSide), temp)
			if err != nil {
				return err
			}
			vr = *temp
		default:
			return RuleSettingError.MODIFER_SIDE_INVALID
		}

		for _, key := range selectWhat {
			if ds := vr.Select(key); ds != nil {
				value, _ := strconv.ParseInt(ds.Value, 10, 64)
				modifer = modifer + value
			}
		}
		reflect.Indirect(reflect.ValueOf(rqr)).FieldByName(rm.TargetField).SetInt(modifer)
		return nil
	}

	return RuleSettingError.UNSUPPORTED_OPERATION
}

func (re *ruleEngine) applyModiferJumpReturn(rqr interface{}, rm Modifer) (bool, error) {
	id := rm.LeftSide
	start, err := strconv.ParseInt(rm.RightSide, 10, 64)
	if err != nil {
		return false, RuleSettingError.MODIFER_SIDE_INVALID
	}

	rsn, err := re.sp.FetchRuleSettings(id, int(start))
	if err != nil {
		return false, RuleSettingError.UNABLE_TO_FETCH
	}
	return re.ApplySettings(rqr, rsn)
}

func (re *ruleEngine) compareDayOfWeek(rqr interface{}, c Condition) (bool, error) {
	var vl reflect.Value
	switch c.LeftType {
	case ConditionSideType.FIELD:
		vl = reflect.ValueOf(rqr).FieldByName(c.LeftSide)
	case ConditionSideType.VALUE:
		temp, err := strconv.ParseInt(c.LeftSide, 10, 64)
		if err != nil {
			return false, err
		}
		vl = reflect.ValueOf(time.Unix(temp, 0))
	default:
		return false, RuleSettingError.CONDITION_SIDE_INVALID
	}
	dl := vl.Interface().(time.Time).Weekday()

	// handle In array
	if c.Compare == RuleConditionCompare.IN || c.Compare == RuleConditionCompare.NOT_IN {
		var vr []string
		switch c.RightType {
		case ConditionSideType.FIELD:
			vr = strings.Split(reflect.ValueOf(rqr).FieldByName(c.RightSide).String(), ",")
		case ConditionSideType.VALUE:
			vr = strings.Split(c.RightSide, ",")
		default:
			return false, RuleSettingError.CONDITION_SIDE_INVALID
		}

		for _, v := range vr {
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				// TODO: log error
				return false, err
			}

			d := time.Unix(i, 0)
			if d.Weekday() == dl {
				if c.Compare == RuleConditionCompare.IN {
					return true, nil
				}
				if c.Compare == RuleConditionCompare.NOT_IN {
					return false, nil
				}
			}
		}

		if c.Compare == RuleConditionCompare.IN {
			return false, nil
		}
		if c.Compare == RuleConditionCompare.NOT_IN {
			return true, nil
		}
	}

	// handle other cases
	var vr int64
	switch c.RightType {
	case ConditionSideType.FIELD:
		vr = reflect.ValueOf(rqr).FieldByName(c.RightSide).Interface().(time.Time).Unix()
	case ConditionSideType.VALUE:
		temp, err := strconv.ParseInt(c.RightSide, 10, 64)
		if err != nil {
			return false, err
		}
		vr = temp
	default:
		return false, RuleSettingError.CONDITION_SIDE_INVALID
	}

	dr := time.Unix(vr, 0).Weekday()
	switch c.Compare {
	case RuleConditionCompare.EQUAL:
		return dl == dr, nil
	case RuleConditionCompare.NOT:
		return dl != dr, nil
	}
	return false, RuleSettingError.UNSUPPORTED_OPERATION

}

func (re *ruleEngine) compareDate(rqr interface{}, c Condition) (bool, error) {
	var vl reflect.Value
	switch c.LeftType {
	case ConditionSideType.FIELD:
		vl = reflect.ValueOf(rqr).FieldByName(c.LeftSide)
	case ConditionSideType.VALUE:
		temp, err := strconv.ParseInt(c.LeftSide, 10, 64)
		if err != nil {
			return false, err
		}
		vl = reflect.ValueOf(time.Unix(temp, 0))
	default:
		return false, RuleSettingError.CONDITION_SIDE_INVALID
	}
	dl := util.StripTimeDMY(vl.Interface().(time.Time))

	// handle In case
	if c.Compare == RuleConditionCompare.IN || c.Compare == RuleConditionCompare.NOT_IN {
		var vr []string
		switch c.RightType {
		case ConditionSideType.FIELD:
			vr = strings.Split(reflect.ValueOf(rqr).FieldByName(c.RightSide).String(), ",")
		case ConditionSideType.VALUE:
			vr = strings.Split(c.RightSide, ",")
		default:
			return false, RuleSettingError.CONDITION_SIDE_INVALID
		}

		for _, v := range vr {
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				// TODO: log error
				return false, nil
			}

			d := util.StripTimeDMY(time.Unix(i, 0))
			if d == dl {
				if c.Compare == RuleConditionCompare.IN {
					return true, nil
				}
				if c.Compare == RuleConditionCompare.NOT_IN {
					return false, nil
				}
			}
		}
		if c.Compare == RuleConditionCompare.IN {
			return false, nil
		}
		if c.Compare == RuleConditionCompare.NOT_IN {
			return true, nil
		}
	}

	// handle other cases
	var vr int64
	switch c.RightType {
	case ConditionSideType.FIELD:
		vr = reflect.ValueOf(rqr).FieldByName(c.RightSide).Interface().(time.Time).Unix()
	case ConditionSideType.VALUE:
		temp, err := strconv.ParseInt(c.RightSide, 10, 64)
		if err != nil {
			return false, err
		}
		vr = temp
	default:
		return false, RuleSettingError.CONDITION_SIDE_INVALID
	}

	dr := util.StripTimeDMY(time.Unix(vr, 0))
	switch c.Compare {
	case RuleConditionCompare.EQUAL:
		return dl.Sub(dr) == 0, nil
	case RuleConditionCompare.NOT:
		return dl.Sub(dr) != 0, nil
	case RuleConditionCompare.MORE:
		return dl.Sub(dr) > 0, nil
	case RuleConditionCompare.LESS:
		return dl.Sub(dr) < 0, nil
	case RuleConditionCompare.MORE_EQUAL:
		return dl.Sub(dr) >= 0, nil
	case RuleConditionCompare.LESS_EQUAL:
		return dl.Sub(dr) <= 0, nil
	}
	return false, RuleSettingError.UNSUPPORTED_OPERATION
}

func (re *ruleEngine) compareGenericString(rqr interface{}, c Condition) (bool, error) {
	var vl string
	switch c.LeftType {
	case ConditionSideType.FIELD:
		vl = reflect.ValueOf(rqr).FieldByName(c.LeftSide).String()
	case ConditionSideType.VALUE:
		vl = c.LeftSide
	default:
		return false, RuleSettingError.CONDITION_SIDE_INVALID
	}

	var vr string
	switch c.RightType {
	case ConditionSideType.FIELD:
		vr = reflect.ValueOf(rqr).FieldByName(c.RightSide).String()
	case ConditionSideType.VALUE:
		vr = c.RightSide
	default:
		return false, RuleSettingError.CONDITION_SIDE_INVALID
	}

	switch c.Compare {
	case RuleConditionCompare.EQUAL:
		return vl == vr, nil
	case RuleConditionCompare.NOT:
		return vl != vr, nil
	}
	// Will not support string IN compare, ambiguous case
	return false, RuleSettingError.UNSUPPORTED_OPERATION
}

func (re *ruleEngine) compareGenericInt(rqr interface{}, c Condition) (bool, error) {
	var vl int64
	switch c.LeftType {
	case ConditionSideType.FIELD:
		vl = reflect.ValueOf(rqr).FieldByName(c.LeftSide).Int()
	case ConditionSideType.VALUE:
		temp, err := strconv.ParseInt(c.LeftSide, 10, 64)
		if err != nil {
			return false, err
		}
		vl = temp
	default:
		return false, RuleSettingError.CONDITION_SIDE_INVALID
	}

	// handle In array
	if c.Compare == RuleConditionCompare.IN || c.Compare == RuleConditionCompare.NOT_IN {
		dl := vl
		var vr []string
		switch c.RightType {
		case ConditionSideType.FIELD:
			vr = strings.Split(reflect.ValueOf(rqr).FieldByName(c.RightSide).String(), ",")
		case ConditionSideType.VALUE:
			vr = strings.Split(c.RightSide, ",")
		default:
			return false, RuleSettingError.CONDITION_SIDE_INVALID
		}

		for _, v := range vr {
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				// TODO: log error
				return false, err
			}

			if i == dl {
				if c.Compare == RuleConditionCompare.IN {
					return true, nil
				}
				if c.Compare == RuleConditionCompare.NOT_IN {
					return false, nil
				}
			}
		}

		if c.Compare == RuleConditionCompare.IN {
			return false, nil
		}
		if c.Compare == RuleConditionCompare.NOT_IN {
			return true, nil
		}
	}

	// Handle other cases
	var vr int64
	switch c.RightType {
	case ConditionSideType.FIELD:
		vr = reflect.ValueOf(rqr).FieldByName(c.RightSide).Int()
	case ConditionSideType.VALUE:
		temp, err := strconv.ParseInt(c.RightSide, 10, 64)
		if err != nil {
			return false, err
		}
		vr = temp
	default:
		return false, RuleSettingError.CONDITION_SIDE_INVALID
	}

	switch c.Compare {
	case RuleConditionCompare.EQUAL:
		return vl-vr == 0, nil
	case RuleConditionCompare.NOT:
		return vl-vr != 0, nil
	case RuleConditionCompare.MORE:
		return vl-vr > 0, nil
	case RuleConditionCompare.LESS:
		return vl-vr < 0, nil
	case RuleConditionCompare.MORE_EQUAL:
		return vl-vr >= 0, nil
	case RuleConditionCompare.LESS_EQUAL:
		return vl-vr <= 0, nil
	}
	return false, RuleSettingError.UNSUPPORTED_OPERATION
}

func (re *ruleEngine) checkSettingSequence(settings []RuleSetting) bool {
	for idx := 0; idx < len(settings)-1; idx++ {
		if settings[idx].Sequence > settings[idx+1].Sequence {
			return false
		}
	}
	return true
}

// CheckRuleCondition Check if the result fit the condition
func (re *ruleEngine) CheckRuleCondition(rqr interface{}, c Condition) (bool, error) {
	switch c.Type {
	case RuleConditionType.DAY_OF_WEEK:
		return re.compareDayOfWeek(rqr, c)
	case RuleConditionType.DATE:
		return re.compareDate(rqr, c)
	case RuleConditionType.STRING:
		return re.compareGenericString(rqr, c)
	case RuleConditionType.INT:
		return re.compareGenericInt(rqr, c)
	case RuleConditionType.MUST:
		return true, nil
	}
	return false, RuleSettingError.UNSUPPORTED_OPERATION
}

// CreateRuleSettings Insert Rules into db
func (re *ruleEngine) CreateRuleSettings(tx *gorm.DB, rs []RuleSetting) error {
	for _, rule := range rs {
		ruleDB, err := rule.MakeDBObject()
		if err != nil {
			return err
		}
		ruleDB.ID = uuid.NewV4().String()
		err = tx.Table(DB_TABLE_RULE).Create(ruleDB).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// FetchRuleSettings fetch Rule set with id
func (re *ruleEngine) FetchRuleSettings(tx *gorm.DB, id string) ([]RuleSetting, error) {
	var rsdb []RuleSettingDB
	err := tx.Table(DB_TABLE_RULE).Where("rule_id = ?", id).Scan(&rsdb).Error
	if err != nil {
		return nil, err
	}

	rs := make([]RuleSetting, len(rsdb))
	for i, ruledb := range rsdb {
		rule, err := ruledb.MakeObject()
		if err != nil {
			return nil, err
		}
		rs[i] = *rule
	}
	return rs, nil
}
