package main

import (
	"fmt"
	"log"

	"github.com/0xtan3/GoSigmaCH/clickhouse"
)

func main() {
	backend := clickhouse.NewClickhouseBackend()
	yamlData := `
title: Test Rule
detection:
    sel:
        EventID: 4624
    condition: sel
`
	queries, err := backend.Convert(yamlData)
	if err != nil {
		log.Fatal(err)
	}
	for _, q := range queries {
		fmt.Println(q)
	}
}
