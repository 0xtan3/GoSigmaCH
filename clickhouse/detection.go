package clickhouse

import (
	"fmt"
	"strings"

	"github.com/0xtan3/GoSigmaCH/internal/ast"
)

func (b *ClickhouseBackend) convertDetection(rule SigmaRule) (string, error) {
	condStr, ok := rule.Detection["condition"].(string)
	if !ok {
		// some rules might have list of conditions (rare but possible), we simplify to string
		return "", fmt.Errorf("condition must be a string")
	}

	var searchKeys []string
	for k := range rule.Detection {
		if k != "condition" {
			searchKeys = append(searchKeys, k)
		}
	}

	expr, err := ast.ParseCondition(condStr, searchKeys)
	if err != nil {
		return "", err
	}

	whereClause, err := b.evalExpr(expr, rule.Detection)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("SELECT * FROM %s WHERE %s", sanitizeTableName(b.TableName), whereClause), nil
}

func (b *ClickhouseBackend) evalExpr(expr ast.Expr, detection map[string]interface{}) (string, error) {
	switch e := expr.(type) {
	case ast.AndExpr:
		left, err := b.evalExpr(e.Left, detection)
		if err != nil {
			return "", err
		}
		right, err := b.evalExpr(e.Right, detection)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s AND %s", left, right), nil
	case ast.OrExpr:
		left, err := b.evalExpr(e.Left, detection)
		if err != nil {
			return "", err
		}
		right, err := b.evalExpr(e.Right, detection)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s OR %s)", left, right), nil
	case ast.NotExpr:
		inner, err := b.evalExpr(e.Expr, detection)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(NOT %s)", inner), nil
	case ast.IdentExpr:
		val, ok := detection[e.Name]
		if !ok {
			return "", fmt.Errorf("identifier %s not found in detection", e.Name)
		}
		return b.evalSearch(val)
	}
	return "", fmt.Errorf("unknown expression type")
}

func (b *ClickhouseBackend) evalSearch(search interface{}) (string, error) {
	// A search can be a map, or a list of maps/values.
	// In Sigma, a map is an AND of its keys.
	// A list is an OR of its elements.
	switch v := search.(type) {
	case map[string]interface{}:
		var parts []string
		for k, val := range v {
			part, err := b.evalFieldCond(k, val)
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		if len(parts) == 1 {
			return parts[0], nil
		}
		return "(" + strings.Join(parts, " AND ") + ")", nil
	case []interface{}:
		var parts []string
		for _, item := range v {
			part, err := b.evalSearch(item)
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		if len(parts) == 1 {
			return parts[0], nil
		}
		return "(" + strings.Join(parts, " OR ") + ")", nil
	}
	return "", fmt.Errorf("invalid search structure")
}
