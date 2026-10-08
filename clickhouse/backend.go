package clickhouse

type ClickhouseBackend struct {
	TableName      string
	TimestampField string
	FullLogField   string
}

func NewClickhouseBackend() *ClickhouseBackend {
	return &ClickhouseBackend{
		TableName:      "logs",
		TimestampField: "timestamp",
		FullLogField:   "full_log",
	}
}
