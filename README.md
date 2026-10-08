# GoSigmaCH

A native Go library for converting [Sigma](https://sigmahq.io/) rules into ClickHouse SQL queries.

GoSigmaCH is designed to be embedded directly into high-performance Go services such as SIEMs, detection engines, rule pipelines, and log processors.

## Installation

Install GoSigmaCH using:

```bash
go get github.com/0xtan3/GoSigmaCH
```

Then import the ClickHouse backend:

```go
import "github.com/0xtan3/GoSigmaCH/clickhouse"
```

## Quick Start

GoSigmaCH provides a `Convert` method for converting both standard Sigma detection rules and correlation rules into ClickHouse SQL.

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

	// Initialize the ClickHouse backend.
	backend := clickhouse.NewClickhouseBackend()

	// Optional: Override default configuration.
	backend.TableName = "http_logs"
	backend.TimestampField = "event_time"

	// Convert the Sigma rule to ClickHouse SQL.
	queries, err := backend.Convert(rule)
	if err != nil {
		log.Fatalf("Failed to convert rule: %v", err)
	}

	// Execute the generated SQL in ClickHouse.
	for _, query := range queries {
		fmt.Println("Generated SQL:", query)
	}
}
```

Example generated SQL:

```sql
SELECT * FROM http_logs
WHERE (status='404' AND uri ILIKE '%/admin%')
```

## Supported Features

### Sigma Modifiers

Supports standard Sigma modifiers including:

- `|contains`
- `|startswith`
- `|endswith`
- `|re`
- `|cidr`
- `|gt`
- `|gte`
- `|lt`
- `|lte`
- `|all`

### Boolean Logic

Supports complex Sigma condition expressions including:

- `and`
- `or`
- `not`
- `1 of sel*`
- `all of filter_*`
- Nested conditions

### Correlation Rules

Supports Sigma correlation rule types including:

- `event_count`
- `value_count`
- `value_sum`
- `value_avg`
- `temporal`
- `temporal_ordered`

### Security

GoSigmaCH includes protections for generated SQL, including:

- Sanitization of table names
- SQL string escaping
- Regular-expression escaping
- Safe handling of Sigma values

## Usage

GoSigmaCH can be embedded into applications such as:

- SIEM detection engines
- Security analytics platforms
- Rule execution pipelines
- Log processing systems
- Threat detection services
- ClickHouse-based security data platforms

## Project Structure

```text
GoSigmaCH/
├── clickhouse/
│   ├── backend.go
│   ├── converter.go
│   ├── correlation.go
│   ├── detection.go
│   └── fieldcond.go
├── go.mod
├── go.sum
└── README.md
```

## Versioning

GoSigmaCH uses semantic versioning.

The latest release can be installed with:

```bash
go get github.com/0xtan3/GoSigmaCH
```

To use a specific release, pin the desired version:

```bash
go get github.com/0xtan3/GoSigmaCH@v0.1.1
```

Available releases are published through the repository's GitHub Releases.

## License

See the repository license for details.