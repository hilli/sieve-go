package token

import "testing"

func TestIsReserved(t *testing.T) {
	for _, w := range []string{"require", "if", "elsif", "else", "stop", "true", "false", "not", "anyof", "allof"} {
		if !IsReserved(w) {
			t.Errorf("expected %q reserved", w)
		}
	}
	for _, w := range []string{"address", "header", "fileinto", "keep", "REQUIRE", ""} {
		if IsReserved(w) {
			t.Errorf("expected %q NOT reserved", w)
		}
	}
}
