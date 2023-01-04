package git

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kballard/go-shellquote"
)

/*
NewTemporaryFileEditor creates a new Editor instance, which can be used to launch a temporary file (containing the content that is passed into NewTemporaryFileEditor)
with the user's default IDE.
Accepts the name for the temporary file, and the file content.
The name of the file will have a random string appended, or if an empty string is provided, the name will just be that random string. If the filename string includes a "*", the random string replaces the last "*".
*/
func NewTemporaryFileEditor(filename string, content []byte) (editor *Editor, err error) {
	temporaryFile, err := createTemporaryFile(filename, content)
	if err != nil {
		return nil, err
	}

	return newEditor(temporaryFile, true)
}

/*
NewEditor creates a new Editor instance, which can be used to launch the file with the user's default IDE.
Accepts the name for the file.
*/
func NewEditor(file string) (editor *Editor, err error) {
	return newEditor(file, false)
}

func newEditor(file string, usesTemporaryFile bool) (editor *Editor, err error) {
	defaultIDE, err := getDefaultGitEditor()
	if err != nil {
		return nil, err
	}

	editor = &Editor{
		Program:           defaultIDE,
		File:              file,
		usesTemporaryFile: usesTemporaryFile,
	}

	return
}

/*
Editor can be used to allow the user to directly edit content using
their default IDE (set via `git var GIT_EDITOR`).
*/
type Editor struct {
	Program           string
	File              string
	Launch            func() error
	usesTemporaryFile bool
}

/*
EditContent launches git's default IDE and waits for the temporary file to be
saved & closed before returning the edited file content.
*/
func (e *Editor) EditContent() ([]byte, error) {
	if err := e.openAndWaitForTextEditor(); err != nil {
		return nil, err
	}

	if e.usesTemporaryFile {
		defer cleanup(e.File)
	}

	return readFile(e.File)
}

func (e *Editor) openAndWaitForTextEditor() error {
	editCmd, err := e.getOpenEditorCommand()
	if err != nil {
		return err
	}

	fmt.Println("Waiting for your editor to close the file...")
	if err := editCmd.Run(); err != nil {
		return err
	}

	return nil
}

// Returns the command we can run to open the default git editor.
func (e *Editor) getOpenEditorCommand() (*exec.Cmd, error) {
	programArgs, err := shellquote.Split(e.Program)
	if err != nil {
		return nil, err
	}

	editCmd := exec.Command(programArgs[0], programArgs[1:]...)
	editCmd.Stdout = os.Stdout
	editCmd.Stdin = os.Stdin
	editCmd.Args = append(editCmd.Args, e.File)
	return editCmd, nil
}

// Get's the OS default git editor via the `git var GIT_EDITOR` command.
func getDefaultGitEditor() (string, error) {
	output, err := exec.Command("git", "var", "GIT_EDITOR").Output()
	if err != nil {
		// Most likely git is not installed, fallback to default OS editor
		return getDefaultEditorByOS()
	}

	editor := os.ExpandEnv(getFirstLine(string(output)))

	if len(editor) < 1 {
		// No default git editor, fallback to default OS editor
		return getDefaultEditorByOS()
	}

	return editor, nil
}

func getFirstLine(output string) string {
	if i := strings.Index(output, "\n"); i >= 0 {
		return output[0:i]
	}
	return output
}

func getDefaultEditorByOS() (string, error) {
	switch runtime.GOOS {
	case "darwin", "linux":
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		return editor, nil
	case "windows":
		// As far as I can tell, Windows doesn't have an easily accesible or
		// comparable option to $EDITOR, so default to notepad for now
		return "notepad", nil
	default:
		return "", fmt.Errorf("unsupported platform")
	}
}

/*
Creates a temporary file containing the data passed. The name of the file will have a random string appended,
or if an empty string is provided, the name will just be that random string. Does not delete the temporary file,
you MUST call os.Remove(returnedFilename) when you are done.

Uses the default directory for temporary files, as returned by TempDir.
*/
func createTemporaryFile(name string, data []byte) (string, error) {
	f, err := os.CreateTemp("", name)
	if err != nil {
		return "", err
	}

	if _, err := f.Write(data); err != nil {
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

func readFile(name string) ([]byte, error) {
	newFixtureData, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return newFixtureData, nil
}

func cleanup(name string) {
	os.Remove(name)
}
