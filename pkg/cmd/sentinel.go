package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// sentinelBegin and sentinelEnd mark the completion configuration block
// in shell config files (~/.zshrc, ~/.bashrc, ~/.bash_profile). This allows
// safe idempotent install/uninstall without corrupting the user's existing config.
const (
	sentinelBegin = "# begin stripe-completion -- managed by stripe cli, do not edit"
	sentinelEnd   = "# end stripe-completion"
)

// computeAddSentinel returns the new file content with the sentinel block added
// or replaced. It performs no I/O. If both markers are present in the correct
// order, the existing block is replaced. If markers are absent, orphaned, or
// reversed, a new block is appended to the content.
func computeAddSentinel(content, line string) string {
	block := fmt.Sprintf("%s\n%s\n%s", sentinelBegin, line, sentinelEnd)

	beginIdx := strings.Index(content, sentinelBegin)
	endIdx := strings.Index(content, sentinelEnd)
	if beginIdx >= 0 && endIdx >= 0 && endIdx > beginIdx {
		end := endIdx + len(sentinelEnd)
		// Include trailing newline if present
		if end < len(content) && content[end] == '\n' {
			end++
		}
		return content[:beginIdx] + block + "\n" + content[end:]
	}

	// Append sentinel block
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + block + "\n"
}

// computeRemoveSentinel returns the new file content with the sentinel block
// removed and a boolean indicating whether a block was found and removed. It
// performs no I/O. If markers are absent, orphaned, or reversed, the content
// is returned unchanged with false.
func computeRemoveSentinel(content string) (string, bool) {
	beginIdx := strings.Index(content, sentinelBegin)
	endIdx := strings.Index(content, sentinelEnd)
	if beginIdx < 0 || endIdx < 0 || endIdx <= beginIdx {
		return content, false
	}

	end := endIdx + len(sentinelEnd)
	// Include trailing newline if present
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return content[:beginIdx] + content[end:], true
}

// readConfigFile opens the file at path, reads its contents, and returns the
// content string along with the file's permission bits. The file is opened once
// and stat'd on the same file descriptor to avoid TOCTOU races. If the file
// does not exist, ("", 0644, nil) is returned.
func readConfigFile(path string) (string, os.FileMode, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0644, nil
		}
		return "", 0, fmt.Errorf("reading %s: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", 0, fmt.Errorf("reading %s: %w", path, err)
	}
	perm := info.Mode().Perm()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", 0, fmt.Errorf("reading %s: %w", path, err)
	}

	return string(data), perm, nil
}

// atomicWriteFile writes data to path atomically by creating a temporary file
// in the same directory, syncing, and renaming over the destination. This
// avoids partial writes visible to concurrent readers. On any error after the
// temp file is created, the temp file is removed.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".stripe-*")
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	tmpName := tmp.Name()

	// Ensure cleanup on any error path after file creation.
	var writeErr error
	defer func() {
		if writeErr != nil {
			os.Remove(tmpName)
		}
	}()

	if _, writeErr = tmp.Write(data); writeErr != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", path, writeErr)
	}
	if writeErr = tmp.Sync(); writeErr != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", path, writeErr)
	}
	if writeErr = tmp.Close(); writeErr != nil {
		return fmt.Errorf("writing %s: %w", path, writeErr)
	}
	if writeErr = os.Chmod(tmpName, perm); writeErr != nil {
		return fmt.Errorf("writing %s: %w", path, writeErr)
	}
	if writeErr = os.Rename(tmpName, path); writeErr != nil {
		return fmt.Errorf("writing %s: %w", path, writeErr)
	}
	return nil
}

// addSentinelBlock adds or replaces a sentinel-delimited block in the given
// config file. If the file does not exist, it is created with mode 0644.
// Existing file permissions are preserved. The operation is idempotent:
// calling it twice with the same line produces the same result as calling
// it once. If the file contains orphaned or reversed markers, a new block
// is appended rather than attempting to repair the malformed state.
func addSentinelBlock(configPath, line string) error {
	content, perm, err := readConfigFile(configPath)
	if err != nil {
		return err
	}

	newContent := computeAddSentinel(content, line)
	return atomicWriteFile(configPath, []byte(newContent), perm)
}

// removeSentinelBlock removes the sentinel-delimited block from the given
// config file. If the file does not exist, this is a no-op. If the markers
// are orphaned or reversed, the file is left unchanged. Existing file
// permissions are preserved.
func removeSentinelBlock(configPath string) error {
	content, perm, err := readConfigFile(configPath)
	if err != nil {
		return err
	}

	// If the file did not exist, readConfigFile returns ("", 0644, nil).
	// computeRemoveSentinel("") returns ("", false), so !found handles both
	// the missing-file case and the no-block-present case uniformly.
	newContent, found := computeRemoveSentinel(content)
	if !found {
		return nil
	}

	return atomicWriteFile(configPath, []byte(newContent), perm)
}

// manualRemnant represents a line in a shell config file that references the
// completion script but is outside our sentinel-managed block.
type manualRemnant struct {
	lineNumber int    // 1-based, for display in user-facing warnings
	lineText   string // trimmed content of the matching line
}

// findManualRemnants scans a shell config file for lines referencing the
// completion script filename that are outside our sentinel block. This detects
// manually-added source/load lines that the user may need to clean up.
//
// Lines inside the sentinel block, blank lines, and comment lines (starting
// with #) are excluded from the scan. Returns nil if the file cannot be read
// or no matches are found.
func findManualRemnants(configPath, scriptFilename string) []manualRemnant {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var remnants []manualRemnant
	inSentinelBlock := false

	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, sentinelBegin) {
			inSentinelBlock = true
			continue
		}
		if strings.Contains(trimmed, sentinelEnd) {
			inSentinelBlock = false
			continue
		}

		if inSentinelBlock {
			continue
		}

		// Skip blank lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.Contains(trimmed, scriptFilename) {
			remnants = append(remnants, manualRemnant{
				lineNumber: i + 1,
				lineText:   trimmed,
			})
		}
	}

	return remnants
}
