package assets

import (
	"strings"
	"testing"
)

func TestLogo(t *testing.T) {
	out := Logo(16, 8, 40)
	if !strings.HasPrefix(out, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" x="16" y="8" width="40" height="40">`) {
		t.Errorf("logo root tag not resized: %.120q", out)
	}
	if strings.Count(out, logoSize) != 0 {
		t.Error("logo still contains the original size")
	}
}
