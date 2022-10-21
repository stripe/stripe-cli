package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kballard/go-shellquote"
)

/*
NewEditor creates a new Editor instance, which can be used to launch a file
(containing the content that is passed into NewEditor) with the OS's
default IDE, and return the new file content after the user saves & closes the file.
*/
func NewEditor(content []byte) (editor *Editor, err error) {
	defaultIDE, err := getDefaultEditor()
	if err != nil {
		return nil, err
	}

	filename, err := createTemporaryFile("tmp", content)
	if err != nil {
		return nil, err
	}

	editor = &Editor{
		Program: defaultIDE,
		File:    filename,
	}

	return
}

/*
Editor can be used to allow the user to directly edit content using
their default IDE (set via `git var GIT_EDITOR`).
*/
type Editor struct {
	Program string
	File    string
	Launch  func() error
}

/*
EditContent launches git's default IDE and waits for the temporary file to be
saved & closed before returning the edited file content.
*/
func (e *Editor) EditContent() ([]byte, error) {
	if err := e.openAndWaitForTextEditor(); err != nil {
		return nil, err
	}

	defer cleanup(e.File)

	return readTemporaryFile(e.File)
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
func getDefaultEditor() (string, error) {
	output, err := exec.Command("git", "var", "GIT_EDITOR").Output()
	if err != nil {
		return "", err
	}

	editor := os.ExpandEnv(getFirstLine(string(output)))

	if len(editor) < 1 {
		return "", errors.New("no default editor found. Please set your GIT_EDITOR var: https://git-scm.com/docs/git-var")
	}

	return editor, nil
}

func getFirstLine(output string) string {
	if i := strings.Index(output, "\n"); i >= 0 {
		return output[0:i]
	}
	return output
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

func readTemporaryFile(name string) ([]byte, error) {
	newFixtureData, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return newFixtureData, nil
}

func cleanup(name string) {
	os.Remove(name)
}
