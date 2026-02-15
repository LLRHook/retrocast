package gateway

import (
	"encoding/json"
	"strconv"
)

// mustMarshal marshals v to json.RawMessage, panicking on error.
// Only for statically-known types that cannot fail.
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic("gateway: mustMarshal: " + err.Error())
	}
	return data
}

// parseSnowflake parses a string snowflake ID to int64.
func parseSnowflake(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
