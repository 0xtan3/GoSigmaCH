package clickhouse

import (
	"io"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type SigmaRule struct {
	Title       string                 `yaml:"title"`
	Name        string                 `yaml:"name"`
	Status      string                 `yaml:"status"`
	Logsource   map[string]interface{} `yaml:"logsource"`
	Detection   map[string]interface{} `yaml:"detection"`
	Correlation *Correlation           `yaml:"correlation"`
}

type Correlation struct {
	Type      string      `yaml:"type"`
	Rules     interface{} `yaml:"rules"` // string or []string
	Timespan  string      `yaml:"timespan"`
	Condition map[string]interface{} `yaml:"condition"`
	GroupBy   []string    `yaml:"group-by"`
}


var tableRegex = regexp.MustCompile(`^[a-zA-Z0-9_\.]+$`)

func sanitizeIdentifier(name string) string {
	if regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(name) {
		return name
	}
	return "`" + name + "`"
}

func sanitizeTableName(name string) string {
	if !tableRegex.MatchString(name) {
		panic("invalid table name")
	}
	return name
}

func escapeString(s string) string {
	// Escape single quotes
	s = strings.ReplaceAll(s, "'", "''")
	// Escape backslashes for SQL string literal
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return s
}

func escapeRegex(s string) string {
	// Fix Regex null byte escaping gap and escape for ClickHouse match()
	s = strings.ReplaceAll(s, "\x00", "\\x00")
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "''")
	return s
}

func (b *ClickhouseBackend) Convert(yamlData string) ([]string, error) {
	dec := yaml.NewDecoder(strings.NewReader(yamlData))
	var rules []SigmaRule
	for {
		var r SigmaRule
		err := dec.Decode(&r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	var queries []string
	for _, rule := range rules {
		if rule.Correlation != nil {
			q, err := b.convertCorrelation(rule, rules)
			if err != nil {
				return nil, err
			}
			queries = append(queries, q)
		} else if rule.Detection != nil {
			q, err := b.convertDetection(rule)
			if err != nil {
				return nil, err
			}
			queries = append(queries, q)
		}
	}

	return queries, nil
}
