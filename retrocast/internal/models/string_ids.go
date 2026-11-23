package models

import (
	"encoding/json"
	"strconv"
)

// StringIDs encodes snowflakes without losing precision in JavaScript clients.
type StringIDs []int64

func (ids StringIDs) MarshalJSON() ([]byte, error) {
	values := make([]string, len(ids))
	for i, id := range ids {
		values[i] = strconv.FormatInt(id, 10)
	}
	return json.Marshal(values)
}
func (ids *StringIDs) UnmarshalJSON(data []byte) error {
	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	result := make(StringIDs, len(values))
	for i, value := range values {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		result[i] = id
	}
	*ids = result
	return nil
}
