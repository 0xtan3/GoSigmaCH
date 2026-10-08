# GoSigmaCH

A native Go library to effortlessly convert Sigma rules into ClickHouse SQL queries.

This module is designed to be embedded directly into high-performance Go services (like SIEMs, rule pipelines, or log processors).

## Installation

To use this in your Go service, you will need to push this code to a Git repository (e.g., `github.com/0xtan3/GoSigmaCH`). Then you can install it effortlessly:

```bash
go get github.com/0xtan3/GoSigmaCH
```

## Quick Start

Import the package and use the `Convert` method. It handles both standard detection rules and correlation rules.

```go
package main

import (
	"fmt"
	"log"
	
	"github.com/0xtan3/GoSigmaCH/clickhouse"
)

func main() {
	rule := `
title: Simple Web Attack
logsource:
    category: webserver
detection:
    selection:
        status: 404
        uri|contains: '/admin'
    condition: selection
`

	// 1. Initialize the backend
	backend := clickhouse.NewClickhouseBackend()
	
	// Optional: Override defaults
	backend.TableName = "http_logs"
	backend.TimestampField = "event_time"

	// 2. Convert the YAML rule
	queries, err := backend.Convert(rule)
	if err != nil {
		log.Fatalf("Failed to convert rule: %v", err)
	}

	// 3. Execute the SQL in ClickHouse
	for _, query := range queries {
		fmt.Println("Generated SQL:", query)
		// Output: SELECT * FROM http_logs WHERE (status='404' AND uri ILIKE '%/admin%')
	}
}
```

## Supported Features

* **All standard Sigma Modifiers**: `|contains`, `|startswith`, `|endswith`, `|re`, `|cidr`, `|gt`, `|gte`, `|lt`, `|lte`, `|all`
* **Boolean logic**: Full support for complex condition statements (`1 of sel*`, `all of filter_*`, `not`, `and`, `or`)
* **Correlation Rules**: Supports Sigma correlation types including `event_count`, `value_count`, `value_sum`, `value_avg`, `temporal`, and `temporal_ordered`.
* **Security First**: Automatically sanitizes table names and strictly escapes regular expressions and strings to prevent SQL Injection.
