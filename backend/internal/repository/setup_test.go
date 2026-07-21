package repository

import "testing"

func TestSetupSiteNameJSONEscapesUserInput(t *testing.T) {
	if got := setupSiteNameJSON(`A "quoted" site`); got != `"A \"quoted\" site"` {
		t.Fatalf("setupSiteNameJSON() = %q", got)
	}
}
