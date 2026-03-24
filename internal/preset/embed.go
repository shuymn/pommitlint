package preset

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

var (
	//go:embed preset.json
	embeddedPreset []byte
	loadOnce       = sync.OnceValues(loadEmbedded)
)

func Load() (Schema, error) {
	return loadOnce()
}

func loadEmbedded() (Schema, error) {
	var schema Schema
	if err := json.Unmarshal(embeddedPreset, &schema); err != nil {
		return Schema{}, fmt.Errorf("decode embedded preset: %w", err)
	}

	if schema.Version != SchemaVersion {
		return Schema{}, fmt.Errorf("unsupported preset schema version: %d", schema.Version)
	}

	return schema, nil
}
