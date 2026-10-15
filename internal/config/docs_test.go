package config

import (
	"reflect"
	"strings"
	"testing"
)

// Every config key must carry a toml name and a doc description, and map to an
// environment variable: the generated docs and shipped config file depend on it.
func TestEveryKeyIsDocumentedAndMappedToEnv(t *testing.T) {
	t.Parallel()
	rt := reflect.TypeOf(Config{})
	for i := range rt.NumField() {
		sec := rt.Field(i)
		section := sec.Tag.Get("toml")
		if section == "" {
			t.Errorf("section %s has no toml tag", sec.Name)
			continue
		}
		for j := range sec.Type.NumField() {
			f := sec.Type.Field(j)
			key := f.Tag.Get("toml")
			if key == "" || f.Tag.Get("doc") == "" {
				t.Errorf("%s.%s needs toml and doc tags", sec.Name, f.Name)
				continue
			}
			env := EnvPrefix + strings.ToUpper(section) + "_" + strings.ToUpper(key)
			if _, ok := envKeys[env]; !ok {
				t.Errorf("%s.%s has no environment mapping %s", sec.Name, f.Name, env)
			}
		}
	}
}
