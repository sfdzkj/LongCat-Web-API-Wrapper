package main

import (
	"encoding/json"
	"testing"
)

func TestParse(t *testing.T) {
	data := []byte(`{
		"taskResult": [
			{
				"extra": [
					{
						"type": "image",
						"content": [
							{
								"url": "https://image.url/test.png",
								"status": "SUCCESS"
							}
						]
					}
				]
			}
		]
	}`)
	var dataObj map[string]any
	if err := json.Unmarshal(data, &dataObj); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	taskResult, ok := dataObj["taskResult"].([]any)
	if !ok || len(taskResult) == 0 {
		t.Fatalf("taskResult missing or invalid")
	}
}
