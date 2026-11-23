package gateway

import (
	"encoding/json"
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
