package prompt

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"

	"github.com/stripe/stripe-cli/pkg/coop/tui"
)

func Select[T comparable](title string, options []huh.Option[T], value *T) error {
	if len(options) == 0 {
		return fmt.Errorf("no options available")
	}
	height := len(options)
	if height < 3 {
		height = 3
	}
	if height > 10 {
		height = 10
	}

	selectField := huh.NewSelect[T]().
		Title(title).
		Options(options...).
		Height(height).
		Value(value).
		WithTheme(tui.HuhTheme())

	var err error
	if Accessible() {
		err = huh.NewForm(huh.NewGroup(selectField)).
			WithTheme(tui.HuhTheme()).
			WithAccessible(true).
			Run()
	} else {
		err = selectField.Run()
	}
	if err == nil {
		return nil
	}
	if errors.Is(err, huh.ErrUserAborted) {
		return fmt.Errorf("canceled")
	}
	return err
}

func Accessible() bool {
	switch strings.ToLower(os.Getenv("STRIPE_COOP_ACCESSIBLE_PROMPTS")) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
