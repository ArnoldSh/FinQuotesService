package tools

import (
	"encoding/json"
	"os"
)

func LoadSupportedCurrencies(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pairs []string
	if err := json.Unmarshal(data, &pairs); err != nil {
		return nil, err
	}
	supported := make(map[string]bool, len(pairs))
	for _, p := range pairs {
		supported[p] = true
	}
	return supported, nil
}
