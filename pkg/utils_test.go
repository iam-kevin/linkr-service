package linkr

import (
	"testing"
	"time"
)

func TestCovertStringDurationToSeconds(t *testing.T) {
	input := "12h"
	got, err := ConvertStringDurationToSeconds(input)
	want := (12 * time.Hour)

	if err != nil {
		t.Error(err)
	}

	if got != want {
		t.Errorf("failed to convert string '%s' to proper seconds. got %v, want %v", input, got, want)
	}
}
