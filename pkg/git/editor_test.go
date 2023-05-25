package git

import (
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileEditor(t *testing.T) {
	t.Run("Creates default Editor", func(t *testing.T) {
		editor, err := NewEditor("")
		assert.NotNil(t, editor)
		assert.Nil(t, err)
	})
}

func TestNewTemporaryFileEditor(t *testing.T) {
	t.Run("Creates default temporary file Editor", func(t *testing.T) {
		editor, err := NewTemporaryFileEditor("", nil)
		assert.NotNil(t, editor)
		assert.Nil(t, err)
	})

	t.Run("creates temporary file with custom name", func(t *testing.T) {
		editor, _ := NewTemporaryFileEditor("foo", nil)

		f, err := os.Stat(editor.File)
		assert.Nil(t, err)
		assert.Contains(t, f.Name(), "foo")
	})
}

func TestGetOpenEditorCommand(t *testing.T) {
	t.Run("with default system editor", func(t *testing.T) {
		editor, err := NewTemporaryFileEditor("", nil)
		assert.NotNil(t, editor)
		assert.Nil(t, err)

		command, _ := editor.getOpenEditorCommand()

		assert.GreaterOrEqual(t, len(command.Args), 2)
		assert.Contains(t, command.Args[len(command.Args)-1], editor.File)
	})

	t.Run("with custom set editor", func(t *testing.T) {
		prevGitEditor, _ := getDefaultGitEditor()
		defer setEditorTo(prevGitEditor)

		setEditorTo("command with multiple --options")

		editor, err := NewTemporaryFileEditor("", nil)
		assert.NotNil(t, editor)
		assert.Nil(t, err)

		command, _ := editor.getOpenEditorCommand()

		assert.Equal(t, len(command.Args), 5)
		assert.Contains(t, command.Args[len(command.Args)-1], editor.File)
	})
}

func TestGetDefaultGitEditor(t *testing.T) {
	t.Run("common default editors", func(t *testing.T) {
		prevGitEditor, _ := getDefaultGitEditor()
		defer setEditorTo(prevGitEditor)

		for _, e := range [4]string{"subl -n -w", "vi", "code --wait", "mate -w"} {
			setEditorTo(e)

			defaultIDE, _ := getDefaultGitEditor()
			assert.Equal(t, defaultIDE, e)
		}
	})

	t.Run("expands env var", func(t *testing.T) {
		prevGitEditor, _ := getDefaultGitEditor()
		defer setEditorTo(prevGitEditor)

		os.Setenv("STRIPE_CLI_TEST_GIT_EDITOR", "value")
		defer os.Unsetenv("STRIPE_CLI_TEST_GIT_EDITOR")

		setEditorTo("$STRIPE_CLI_TEST_GIT_EDITOR")

		defaultIDE, _ := getDefaultGitEditor()
		assert.Equal(t, defaultIDE, "value")
	})

	t.Run("no GIT_EDITOR falls back to EDITOR or OS fallback", func(t *testing.T) {
		prevGitEditor, _ := getDefaultGitEditor()
		defer setEditorTo(prevGitEditor)

		setEditorTo("")

		newEditor, _ := getDefaultGitEditor()
		if runtime.GOOS == "windows" {
			assert.Equal(t, "notepad", newEditor)
		} else {
			expectedEditor := os.Getenv("EDITOR")
			if expectedEditor == "" {
				expectedEditor = "vi"
			}

			assert.Equal(t, expectedEditor, newEditor)
		}
	})
}

func TestGetFirstLine(t *testing.T) {
	assert.Equal(t, "abc", getFirstLine("abc\n123"))
	assert.Equal(t, "abc", getFirstLine("abc"))
	assert.Equal(t, "abc", getFirstLine(
		`abc
`,
	))
	assert.Equal(t, "abc", getFirstLine("abc\n\n123\n\n\n"))
}

func setEditorTo(newTemporaryFileEditor string) {
	exec.Command("git", "config", "--global", "core.editor", newTemporaryFileEditor).Run()
}
