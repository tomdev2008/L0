package common

import (
	"fmt"
)

// ResponseData is the response data.
type ResponseData struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
}

// Err returns error.
func (d *ResponseData) Err() error {
	if d.Error == "" {
		return nil
	}
	return fmt.Errorf(d.Error)
}

// StringResult returns string format result.
func (d *ResponseData) StringResult() string {
	if ret, ok := d.Result.(string); ok {
		return ret
	}
	return ""
}

// MapResult returns map format result.
func (d *ResponseData) MapResult() map[string]interface{} {
	if ret, ok := d.Result.(map[string]interface{}); ok {
		return ret
	}
	return map[string]interface{}{}
}
