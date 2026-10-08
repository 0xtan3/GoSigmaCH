package clickhouse

import (
	"fmt"
	"strings"
)

func (b *ClickhouseBackend) evalFieldCond(fieldWithMods string, value interface{}) (string, error) {
	parts := strings.Split(fieldWithMods, "|")
	field := parts[0]
	mods := parts[1:]

	field = sanitizeIdentifier(field)

	isAll := false
	isStartswith := false
	isEndswith := false
	isContains := false
	isRe := false
	isGt, isGte, isLt, isLte := false, false, false, false
	isCidr := false
	isExists := false

	for _, m := range mods {
		switch m {
		case "all":
			isAll = true
		case "startswith":
			isStartswith = true
		case "endswith":
			isEndswith = true
		case "contains":
			isContains = true
		case "re":
			isRe = true
		case "gt":
			isGt = true
		case "gte":
			isGte = true
		case "lt":
			isLt = true
		case "lte":
			isLte = true
		case "cidr":
			isCidr = true
		case "exists":
			isExists = true
		}
	}

	if isExists {
		valBool, _ := value.(bool)
		if valBool {
			return fmt.Sprintf("isNotNull(%s)", field), nil
		}
		return fmt.Sprintf("isNull(%s)", field), nil
	}

	if value == nil {
		return fmt.Sprintf("isNull(%s)", field), nil
	}

	if field == "keywords" {
		field = b.FullLogField
		field = sanitizeIdentifier(field)
		// For keywords we use hasToken
		valStr := fmt.Sprintf("%v", value)
		return fmt.Sprintf("hasToken(%s, '%s')", field, escapeString(valStr)), nil
	}

	var values []interface{}
	switch v := value.(type) {
	case []interface{}:
		values = v
	default:
		values = []interface{}{v}
	}

	var conds []string
	for _, val := range values {
		valStr := fmt.Sprintf("%v", val)
		var c string
		if isRe {
			c = fmt.Sprintf("match(%s, '%s')", field, escapeRegex(valStr))
		} else if isStartswith {
			valStr = strings.ReplaceAll(valStr, "%", "\\%")
			valStr = strings.ReplaceAll(valStr, "_", "\\_")
			c = fmt.Sprintf("%s ILIKE '%s%%'", field, escapeString(valStr))
		} else if isEndswith {
			valStr = strings.ReplaceAll(valStr, "%", "\\%")
			valStr = strings.ReplaceAll(valStr, "_", "\\_")
			c = fmt.Sprintf("%s ILIKE '%%%s'", field, escapeString(valStr))
		} else if isContains {
			valStr = strings.ReplaceAll(valStr, "%", "\\%")
			valStr = strings.ReplaceAll(valStr, "_", "\\_")
			c = fmt.Sprintf("%s ILIKE '%%%s%%'", field, escapeString(valStr))
		} else if isGt {
			c = fmt.Sprintf("%s > %s", field, valStr) // assuming numeric
		} else if isGte {
			c = fmt.Sprintf("%s >= %s", field, valStr)
		} else if isLt {
			c = fmt.Sprintf("%s < %s", field, valStr)
		} else if isLte {
			c = fmt.Sprintf("%s <= %s", field, valStr)
		} else if isCidr {
			c = fmt.Sprintf("isIPAddressInRange(%s, '%s')", field, escapeString(valStr))
		} else {
			// default equals or IN
			if valBool, ok := val.(bool); ok {
				if valBool {
					c = fmt.Sprintf("%s=true", field)
				} else {
					c = fmt.Sprintf("%s=false", field)
				}
			} else {
				// check if it has wildcards
				if strings.Contains(valStr, "*") || strings.Contains(valStr, "?") {
					valStr = strings.ReplaceAll(valStr, "*", "%")
					valStr = strings.ReplaceAll(valStr, "?", "_")
					c = fmt.Sprintf("%s ILIKE '%s'", field, escapeString(valStr))
				} else {
					c = fmt.Sprintf("%s='%s'", field, escapeString(valStr))
				}
			}
		}
		conds = append(conds, c)
	}

	if len(conds) == 1 {
		return conds[0], nil
	}

	joiner := " OR "
	if isAll {
		joiner = " AND "
	}
	
	// If it's all simple equality, use IN
	if !isAll && !isRe && !isStartswith && !isEndswith && !isContains && !isGt && !isGte && !isLt && !isLte && !isCidr {
		allEq := true
		for _, v := range values {
			valStr := fmt.Sprintf("%v", v)
			if strings.Contains(valStr, "*") || strings.Contains(valStr, "?") || strings.Contains(valStr, "%") || strings.Contains(valStr, "_") {
				allEq = false
				break
			}
			if _, ok := v.(bool); ok {
				allEq = false
				break
			}
		}
		if allEq {
			var inVals []string
			for _, v := range values {
				inVals = append(inVals, fmt.Sprintf("'%s'", escapeString(fmt.Sprintf("%v", v))))
			}
			return fmt.Sprintf("%s IN (%s)", field, strings.Join(inVals, ", ")), nil
		}
	}

	return "(" + strings.Join(conds, joiner) + ")", nil
}
