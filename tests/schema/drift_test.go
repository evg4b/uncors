package schema_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schema.json is not consulted at runtime — it exists so editors can complete and
// check a config file. Nothing therefore tells anyone when it drifts from the Go
// structs, which is how it came to bless a `clear-time` that does not exist while
// rejecting the `max-size` that is required.
func TestSchemaMatchesConfigStructs(t *testing.T) {
	content, err := os.ReadFile("../../schema.json")
	require.NoError(t, err)

	var document struct {
		Properties  map[string]json.RawMessage `json:"properties"`
		Definitions map[string]struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"definitions"`
	}

	require.NoError(t, json.Unmarshal(content, &document))

	t.Run("root", func(t *testing.T) {
		// interactive is a CLI flag only, so it is deliberately absent.
		assertSameKeys(t, config.UncorsConfig{}, keysOf(document.Properties))
	})

	definitions := map[string]any{
		"OptionsHandling": config.OptionsHandling{},
		"StaticDirectory": config.StaticDirectory{},
		"Rewrite":         config.RewritingOption{},
		"Script":          config.Script{},
	}

	for name, value := range definitions {
		t.Run(name, func(t *testing.T) {
			definition, ok := document.Definitions[name]
			require.True(t, ok, "schema.json has no %s definition", name)

			assertSameKeys(t, value, keysOf(definition.Properties))
		})
	}

	t.Run("cache-config", func(t *testing.T) {
		var cacheConfig struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}

		require.NoError(t, json.Unmarshal(document.Properties["cache-config"], &cacheConfig))
		assertSameKeys(t, config.CacheConfig{}, keysOf(cacheConfig.Properties))
	})
}

// assertSameKeys compares the yaml keys of a config struct with the properties
// the schema declares for it.
func assertSameKeys(t *testing.T, value any, documented []string) {
	t.Helper()

	assert.Equal(t, yamlKeys(value), documented,
		"schema.json and the config struct describe different keys")
}

// yamlKeys returns the sorted yaml field names of a struct, following inline
// embedding and skipping fields marked `yaml:"-"`.
func yamlKeys(value any) []string {
	var keys []string

	for field := range reflect.TypeOf(value).Fields() {
		tag, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if tag == "-" {
			continue
		}

		if strings.Contains(field.Tag.Get("yaml"), "inline") {
			keys = append(keys, yamlKeys(reflect.New(field.Type).Elem().Interface())...)

			continue
		}

		if tag == "" {
			tag = strings.ToLower(field.Name)
		}

		keys = append(keys, tag)
	}

	sort.Strings(keys)

	return keys
}

func keysOf(properties map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
