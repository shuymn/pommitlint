package preset_test

import (
	"testing"

	"github.com/shuymn/pommitlint/internal/preset"
)

func TestLoadEmbeddedPreset(t *testing.T) {
	t.Parallel()

	got, err := preset.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Version != preset.SchemaVersion {
		t.Fatalf("Version = %d, want %d", got.Version, preset.SchemaVersion)
	}

	if got.Source.ConfigPackage != "@commitlint/config-conventional" {
		t.Fatalf("ConfigPackage = %q", got.Source.ConfigPackage)
	}
}
