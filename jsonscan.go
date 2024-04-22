package dzi_processing

import (
	"encoding/json"
	"fmt"
)

func JsonScan[T any](src interface{}, b T) error {
	switch v := src.(type) {
	case []byte:
		if string(v) == "null" {
			v = []byte("{}")
		}
		return json.Unmarshal(v, b)
	case string:
		return json.Unmarshal([]byte(v), b)
	case nil:
		return nil
	default:
		return fmt.Errorf("cannot convert %T", src)
	}
}
