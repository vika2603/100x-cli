package profile

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vika2603/100x-cli/internal/cmd/factory"
	"github.com/vika2603/100x-cli/internal/output"
)

func TestListEmptyHuman(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stdout := &bytes.Buffer{}
	cmd := newCmdList(&factory.Factory{
		IO: &output.Renderer{Out: stdout, Err: &bytes.Buffer{}, Format: output.FormatHuman},
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "No profiles configured.") {
		t.Fatalf("stdout=%q", got)
	}
}
