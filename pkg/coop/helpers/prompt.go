package helpers

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

type sailPromptTheme struct{}

func (sailPromptTheme) Theme(isDark bool) *huh.Styles {
	lightDark := lipgloss.LightDark(isDark)
	purple500 := lightDark(lipgloss.Color("#4f46d8"), lipgloss.Color("#625afa"))
	purple400 := lightDark(lipgloss.Color("#5f52e8"), lipgloss.Color("#8d7ffa"))
	green400 := lightDark(lipgloss.Color("#2f8506"), lipgloss.Color("#3fa40d"))
	gray300 := lightDark(lipgloss.Color("#2f3640"), lipgloss.Color("#a3acba"))
	gray400 := lightDark(lipgloss.Color("#4f5967"), lipgloss.Color("#87909f"))
	gray500 := lightDark(lipgloss.Color("#697384"), lipgloss.Color("#687385"))
	gray700 := lightDark(lipgloss.Color("#d9dee7"), lipgloss.Color("#414552"))
	border := lipgloss.Blend1D(3, gray500, purple400)[1]
	panel := lightDark(lipgloss.Darken(gray700, 0.04), lipgloss.Lighten(gray700, 0.08))

	styles := huh.ThemeBase(isDark)
	styles.Focused.Base = styles.Focused.Base.BorderForeground(border)
	styles.Focused.Title = styles.Focused.Title.Foreground(purple500).Bold(true)
	styles.Focused.Description = styles.Focused.Description.Foreground(gray400)
	styles.Focused.SelectSelector = styles.Focused.SelectSelector.Foreground(purple500)
	styles.Focused.NextIndicator = styles.Focused.NextIndicator.Foreground(purple400)
	styles.Focused.PrevIndicator = styles.Focused.PrevIndicator.Foreground(purple400)
	styles.Focused.Option = styles.Focused.Option.Foreground(gray300)
	styles.Focused.SelectedOption = styles.Focused.SelectedOption.Foreground(green400)
	styles.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(green400).SetString("▶ ")
	styles.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(gray500).SetString("  ")
	styles.Focused.UnselectedOption = styles.Focused.UnselectedOption.Foreground(gray400)
	styles.Focused.FocusedButton = styles.Focused.FocusedButton.Foreground(lipgloss.Color("#ffffff")).Background(purple500)
	styles.Focused.BlurredButton = styles.Focused.BlurredButton.Foreground(gray300).Background(panel)
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
