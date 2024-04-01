package website

import (
	"encoding/json"
	"log/slog"
)

func convertJSON(data any) []byte {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		slog.Error("Unable to convert JSON.", err)
	}
	return jsonData
}
