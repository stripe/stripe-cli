package git

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEditor(t *testing.T) {
	t.Run("Creates default Editor", func(t *testing.T) {
		editor, err := NewEditor("", nil)
		assert.NotNil(t, editor)
		assert.Nil(t, err)
	})

	t.Run("creates temporary file with custom name", func(t *testing.T) {
		editor, _ := NewEditor("foo", nil)

		f, err := os.Stat(editor.File)
		assert.Nil(t, err)
		assert.Equal(t, f.Name(), "foo")
	})

	t.Run("missing GIT_EDITOR", func(t *testing.T) {
		prevGitEditor := getCurrentEditor()
		defer setEditorTo(prevGitEditor)

		setEditorTo("")

		editor, err := NewEditor("", nil)
		assert.NotNil(t, err)
		assert.Nil(t, editor)
	})
}

func TestGetOpenEditorCommand(t *testing.T) {
	t.Run("with default system editor", func(t *testing.T) {
		editor, err := NewEditor("", nil)
		assert.NotNil(t, editor)
		assert.Nil(t, err)

		command, _ := editor.getOpenEditorCommand()

		assert.GreaterOrEqual(t, len(command.Args), 2)
		assert.Contains(t, command.Args[len(command.Args)-1], editor.File)
	})

	t.Run("with custom set editor", func(t *testing.T) {
		prevGitEditor := getCurrentEditor()
		defer setEditorTo(prevGitEditor)

		setEditorTo("command with multiple --options")

		editor, err := NewEditor("", nil)
		assert.NotNil(t, editor)
		assert.Nil(t, err)

		command, _ := editor.getOpenEditorCommand()

		assert.Equal(t, len(command.Args), 5)
		assert.Contains(t, command.Args[len(command.Args)-1], editor.File)
	})
}

func TestGetDefaultEditor(t *testing.T) {
	t.Run("common default editors", func(t *testing.T) {
		prevGitEditor := getCurrentEditor()
		defer setEditorTo(prevGitEditor)

		for _, e := range [4]string{"subl -n -w", "vi", "code --wait", "mate -w"} {
			setEditorTo(e)

			defaultIDE, _ := getDefaultEditor()
			assert.Equal(t, defaultIDE, e)
		}
	})

	t.Run("expands env var", func(t *testing.T) {
		prevGitEditor := getCurrentEditor()
		defer setEditorTo(prevGitEditor)

		os.Setenv("STRIPE_CLI_TEST_GIT_EDITOR", "value")
		defer os.Unsetenv("STRIPE_CLI_TEST_GIT_EDITOR")

		setEditorTo("$STRIPE_CLI_TEST_GIT_EDITOR")

		defaultIDE, _ := getDefaultEditor()
		assert.Equal(t, defaultIDE, "value")
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

func getCurrentEditor() string {
	e, _ := exec.Command("git", "var", "GIT_EDITOR").Output()
	return string(e)
}

func setEditorTo(newEditor string) {
	exec.Command("git", "config", "--global", "core.editor", newEditor).Run()
}
