package origins

import (
	"errors"
	"fmt"
	"testing"
)

func TestUncategorizedFixturesAreAllowed(t *testing.T) {
	_ = errors.New("fixture")
	_ = fmt.Errorf("fixture: %s", t.Name())
}
