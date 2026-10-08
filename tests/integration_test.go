package tests

import (
	"testing"

	"github.com/0xtan3/GoSigmaCH/clickhouse"
)

func TestBasicDetection(t *testing.T) {
	yamlData := `
title: Test
detection:
    sel:
        fieldA: valueA
        fieldB: valueB
    condition: sel
`
	b := clickhouse.NewClickhouseBackend()
	queries, err := b.Convert(yamlData)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}
	expected := "SELECT * FROM logs WHERE (fieldA='valueA' AND fieldB='valueB')"
	if queries[0] != expected && queries[0] != "SELECT * FROM logs WHERE (fieldB='valueB' AND fieldA='valueA')" {
		t.Errorf("got %s", queries[0])
	}
}

func TestModifiers(t *testing.T) {
	yamlData := `
title: Test
detection:
    sel:
        fieldA|startswith: val
        fieldB|gt: 100
        fieldC|re: ':[^ \\]'
    condition: sel
`
	b := clickhouse.NewClickhouseBackend()
	queries, err := b.Convert(yamlData)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(queries[0])
}

func TestCorrelation(t *testing.T) {
	yamlData := `
title: Base Rule
name: base_rule
detection:
    sel:
        EventID: 1234
    condition: sel
---
title: Event Count Correlation
correlation:
    type: event_count
    rules: base_rule
    timespan: 5m
    condition:
        gte: 10
`
	b := clickhouse.NewClickhouseBackend()
	queries, err := b.Convert(yamlData)
	if err != nil {
		t.Fatal(err)
	}
	for i, q := range queries {
		t.Logf("Query %d: %s", i, q)
	}
}
