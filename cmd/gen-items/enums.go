package main

import (
	"embed"
	"encoding/json"
	"strconv"
)

//go:embed enums/*.json
var enumFS embed.FS

// loadEnum loads a Smithbox param-enum JSON (int key -> English label).
func loadEnum(name string) (map[int]string, error) {
	data, err := enumFS.ReadFile("enums/" + name + ".json")
	if err != nil {
		return nil, err
	}
	var ef struct {
		Options []struct {
			Key   string `json:"Key"`
			Names []struct {
				Language string `json:"Language"`
				Text     string `json:"Text"`
			} `json:"Names"`
		} `json:"Options"`
	}
	if err := json.Unmarshal(data, &ef); err != nil {
		return nil, err
	}
	m := make(map[int]string, len(ef.Options))
	for _, o := range ef.Options {
		k, err := strconv.Atoi(o.Key)
		if err != nil {
			continue
		}
		for _, n := range o.Names {
			if n.Language == "English" {
				m[k] = n.Text
				break
			}
		}
	}
	return m, nil
}
