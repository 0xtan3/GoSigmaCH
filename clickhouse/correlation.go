package clickhouse

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/0xtan3/GoSigmaCH/internal/ast"
)

func (b *ClickhouseBackend) convertCorrelation(rule SigmaRule, allRules []SigmaRule) (string, error) {
	corr := rule.Correlation
	if corr == nil {
		return "", fmt.Errorf("no correlation block")
	}

	var baseQueries []string
	var baseRules []string

	switch v := corr.Rules.(type) {
	case string:
		baseRules = []string{v}
	case []interface{}:
		for _, ri := range v {
			if rStr, ok := ri.(string); ok {
				baseRules = append(baseRules, rStr)
			}
		}
	}

	for _, brName := range baseRules {
		// find rule
		var found SigmaRule
		for _, ar := range allRules {
			if ar.Name == brName {
				found = ar
				break
			}
		}
		if found.Name == "" {
			return "", fmt.Errorf("base rule %s not found", brName)
		}
		
		var searchKeys []string
		for k := range found.Detection {
			if k != "condition" {
				searchKeys = append(searchKeys, k)
			}
		}

		condStr, _ := found.Detection["condition"].(string)
		expr, err := ast.ParseCondition(condStr, searchKeys)
		if err != nil {
			return "", err
		}

		whereClause, err := b.evalExpr(expr, found.Detection)
		if err != nil {
			return "", err
		}

		if len(baseRules) > 1 {
			// multi rule
			baseQueries = append(baseQueries, fmt.Sprintf("SELECT *, '%s' AS sigma_rule_id FROM %s WHERE %s", brName, sanitizeTableName(b.TableName), whereClause))
		} else {
			// single rule
			baseQueries = append(baseQueries, fmt.Sprintf("SELECT * FROM %s WHERE %s", sanitizeTableName(b.TableName), whereClause))
		}
	}

	search := strings.Join(baseQueries, " UNION ALL ")
	
	selectFields := "*"
	if len(corr.GroupBy) > 0 {
		var safeGroup []string
		for _, g := range corr.GroupBy {
			safeGroup = append(safeGroup, sanitizeIdentifier(g))
		}
		selectFields = strings.Join(safeGroup, ", ")
	}

	var aggregate string
	var havingCond string

	condOp, condVal, condField := parseConditionBlock(corr.Condition)

	switch corr.Type {
	case "event_count":
		aggregate = ", count(*) AS event_count"
		havingCond = fmt.Sprintf("event_count %s %d", condOp, condVal)
	case "value_count":
		aggregate = fmt.Sprintf(", uniqExact(%s) AS value_count", sanitizeIdentifier(condField))
		havingCond = fmt.Sprintf("value_count %s %d", condOp, condVal)
	case "value_sum":
		aggregate = fmt.Sprintf(", sum(%s) AS value_sum", sanitizeIdentifier(condField))
		havingCond = fmt.Sprintf("value_sum %s %d", condOp, condVal)
	case "value_avg":
		aggregate = fmt.Sprintf(", avg(%s) AS value_avg", sanitizeIdentifier(condField))
		havingCond = fmt.Sprintf("value_avg %s %d", condOp, condVal)
	case "temporal":
		aggregate = fmt.Sprintf(", uniqExact(sigma_rule_id) AS rule_count, min(%s) AS first_event, max(%s) AS last_event", sanitizeIdentifier(b.TimestampField), sanitizeIdentifier(b.TimestampField))
		havingCond = fmt.Sprintf("rule_count %s %d", condOp, condVal)
	case "temporal_ordered":
		aggregate = fmt.Sprintf(", arrayStringConcat(groupArray(sigma_rule_id), ',') AS rule_sequence, uniqExact(sigma_rule_id) AS rule_count, min(%s) AS first_event, max(%s) AS last_event", sanitizeIdentifier(b.TimestampField), sanitizeIdentifier(b.TimestampField))
		havingCond = fmt.Sprintf("rule_count %s %d", condOp, condVal)
	default:
		return "", fmt.Errorf("unsupported correlation type: %s", corr.Type)
	}

	timespanSec := parseTimespan(corr.Timespan)
	if corr.Type == "temporal" || corr.Type == "temporal_ordered" {
		havingCond += fmt.Sprintf(" AND toUnixTimestamp(last_event) - toUnixTimestamp(first_event) <= %d", timespanSec)
	}

	groupByStr := ""
	if len(corr.GroupBy) > 0 {
		groupByStr = " GROUP BY " + selectFields
	}

	query := fmt.Sprintf("SELECT %s%s FROM (%s) AS subquery%s HAVING %s", selectFields, aggregate, search, groupByStr, havingCond)
	return query, nil
}

func parseConditionBlock(c map[string]interface{}) (string, int, string) {
	op := "="
	val := 0
	field := ""

	if f, ok := c["field"].(string); ok {
		field = f
	}

	for k, v := range c {
		if k == "field" {
			continue
		}
		switch k {
		case "gt":
			op = ">"
		case "gte":
			op = ">="
		case "lt":
			op = "<"
		case "lte":
			op = "<="
		case "eq":
			op = "="
		case "neq":
			op = "!="
		}
		switch valV := v.(type) {
		case int:
			val = valV
		case string:
			val, _ = strconv.Atoi(valV)
		}
	}
	return op, val, field
}

func parseTimespan(ts string) int {
	if ts == "" {
		return 0
	}
	numStr := ""
	unit := ""
	for i, c := range ts {
		if c >= '0' && c <= '9' {
			numStr += string(c)
		} else {
			unit = ts[i:]
			break
		}
	}
	num, _ := strconv.Atoi(numStr)
	switch unit {
	case "s":
		return num
	case "m":
		return num * 60
	case "h":
		return num * 3600
	case "d":
		return num * 86400
	}
	return num
}
