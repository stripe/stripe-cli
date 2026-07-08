package helpers

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/coop/colors"
)

type sailPromptTheme struct{}

func (sailPromptTheme) Theme(isDark bool) *huh.Styles {
	palette := colors.NewPalette(isDark)

	styles := huh.ThemeBase(isDark)
	styles.Focused.Base = styles.Focused.Base.BorderForeground(palette.Border)
	styles.Focused.Title = styles.Focused.Title.Foreground(palette.Purple500).Bold(true)
	styles.Focused.Description = styles.Focused.Description.Foreground(palette.Gray400)
	styles.Focused.SelectSelector = styles.Focused.SelectSelector.Foreground(palette.Purple500)
	styles.Focused.NextIndicator = styles.Focused.NextIndicator.Foreground(palette.Purple400)
	styles.Focused.PrevIndicator = styles.Focused.PrevIndicator.Foreground(palette.Purple400)
	styles.Focused.Option = styles.Focused.Option.Foreground(palette.Gray300)
	styles.Focused.SelectedOption = styles.Focused.SelectedOption.Foreground(palette.Green400)
	styles.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(palette.Green400).SetString("▶ ")
	styles.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(palette.Gray500).SetString("  ")
	styles.Focused.UnselectedOption = styles.Focused.UnselectedOption.Foreground(palette.Gray400)
	styles.Focused.FocusedButton = styles.Focused.FocusedButton.Foreground(palette.OnBrand).Background(palette.Purple500)
	styles.Focused.BlurredButton = styles.Focused.BlurredButton.Foreground(palette.Gray300).Background(palette.Panel)
	styles.Blurred = styles.Focused
	return styles
}

func huhTheme() huh.Theme {
	return sailPromptTheme{}
}

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
		WithTheme(huhTheme())

	var err error
	if Accessible() {
		err = huh.NewForm(huh.NewGroup(selectField)).
			WithTheme(huhTheme()).
			WithAccessible(true).
			Run()
	} else {
		err = selectField.Run()
	}
	if err == nil {
		return nil
	}
	return normalizePromptError(err)
}

func normalizePromptError(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return fmt.Errorf("canceled: %w", err)
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
