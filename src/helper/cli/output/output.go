package output

import (
	"encoding/json"
	"fmt"
)

// Response is a universal wrapper for JSON responses
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// PrintJSON outputs data in JSON format
func PrintJSON(data any) error {
	response := Response{
		Success: true,
		Data:    data,
	}
	return printResponse(response)
}

// PrintJSONError outputs an error in JSON format
func PrintJSONError(err string) error {
	response := Response{
		Success: false,
		Error:   err,
	}
	return printResponse(response)
}

func printResponse(response Response) error {
	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonBytes))
	return nil
}
