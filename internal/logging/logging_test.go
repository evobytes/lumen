package logging_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/evobytes/lumen/internal/logging"
)

func TestNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, level, format string
		wantErr             bool
		wantContains        string
	}{
		{"json info", "info", "json", false, `"msg":"hello"`},
		{"text debug", "debug", "text", false, "msg=hello"},
		{"bad level", "loud", "json", true, ""},
		{"bad format", "info", "xml", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			l, err := logging.New(&buf, tt.level, tt.format)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			l.Info("hello")
			if !strings.Contains(buf.String(), tt.wantContains) {
				t.Errorf("output %q lacks %q", buf.String(), tt.wantContains)
			}
		})
	}
}
