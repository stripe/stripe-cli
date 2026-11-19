package agentsetup

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// executableResolver excludes project-controlled locations, not arbitrary
// untrusted software. Non-project installation directories must be trusted.
// Like ordinary executable lookup, this does not prevent replacement races.
type executableResolver struct {
	dirs       []string
	roots      []string
	extensions []string
	env        []string
	cwd        string
	err        error
}

func newExecutableResolver() *executableResolver {
	r := &executableResolver{env: os.Environ()}
	r.cwd, r.err = os.Getwd()
	if r.err != nil {
		return r
	}
	// Inspect both spellings: a working directory can itself contain aliases.
	resolved, err := filepath.EvalSymlinks(r.cwd)
	if err != nil {
		r.err = err
		return r
	}
	for _, start := range []string{r.cwd, resolved} {
		if !comparableAbsolutePath(start) {
			r.err = &exec.Error{Name: start, Err: exec.ErrNotFound}
			return r
		}
		r.roots = append(r.roots, start)
		for dir := start; ; dir = filepath.Dir(dir) {
			for _, marker := range []string{".git", ".hg", "package.json", "go.mod", "Cargo.toml", "pyproject.toml"} {
				if _, err := os.Lstat(filepath.Join(dir, marker)); err == nil {
					r.roots = append(r.roots, dir)
					target, err := filepath.EvalSymlinks(dir)
					if err != nil {
						r.err = err
						return r
					}
					r.roots = append(r.roots, target)
				} else if !os.IsNotExist(err) {
					r.err = err
					return r
				}
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	var path, pathExt string
	for _, entry := range r.env {
		key, value, _ := strings.Cut(entry, "=")
		if pathKey(key) {
			path = value
		}
		if strings.EqualFold(key, "PATHEXT") {
			pathExt = value
		}
	}
	if runtime.GOOS == "windows" {
		if pathExt == "" {
			pathExt = ".COM;.EXE;.BAT;.CMD"
		}
		for _, ext := range strings.Split(strings.ToLower(pathExt), ";") {
			if ext != "" {
				r.extensions = append(r.extensions, "."+strings.TrimPrefix(ext, "."))
			}
		}
	}
	for _, dir := range filepath.SplitList(path) {
		if r.safe(dir) {
			r.dirs = append(r.dirs, filepath.Clean(dir))
		}
	}
	filtered := make([]string, 0, len(r.env)+1)
	for _, entry := range r.env {
		key, _, _ := strings.Cut(entry, "=")
		if !pathKey(key) {
			filtered = append(filtered, entry)
		}
	}
	filtered = append(filtered, "PATH="+strings.Join(r.dirs, string(os.PathListSeparator)))
	r.env = filtered
	return r
}

func pathKey(key string) bool {
	return key == "PATH" || runtime.GOOS == "windows" && strings.EqualFold(key, "PATH")
}

func pathContains(root, path string) bool {
	if runtime.GOOS == "windows" {
		root, path = windowsComparisonPath(root), windowsComparisonPath(path)
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func windowsComparisonPath(path string) string {
	path = strings.ToLower(strings.ReplaceAll(path, "/", `\`))
	// Extended-length paths and ordinary paths can name the same location.
	if strings.HasPrefix(path, `\\?\unc\`) {
		return `\\` + strings.TrimPrefix(path, `\\?\unc\`)
	}
	return strings.TrimPrefix(path, `\\?\`)
}

func npmBin(path string) bool {
	for dir := filepath.Clean(path); ; dir = filepath.Dir(dir) {
		if strings.EqualFold(filepath.Base(dir), ".bin") && strings.EqualFold(filepath.Base(filepath.Dir(dir)), "node_modules") {
			return true
		}
		if filepath.Dir(dir) == dir {
			return false
		}
	}
}

func comparableAbsolutePath(path string) bool {
	if !filepath.IsAbs(path) {
		return false
	}
	if runtime.GOOS == "windows" {
		// Fail closed on device namespaces that cannot be compared with normal
		// drive-letter or UNC project roots.
		path = strings.ReplaceAll(path, "/", `\`)
		return !strings.HasPrefix(path, `\\.\`) && filepath.IsAbs(windowsComparisonPath(path))
	}
	return true
}

func (r *executableResolver) safe(path string) bool {
	if r.err != nil || !comparableAbsolutePath(path) || npmBin(path) {
		return false
	}
	target, err := filepath.EvalSymlinks(path)
	if err != nil || !comparableAbsolutePath(target) || npmBin(target) {
		return false
	}
	for _, root := range r.roots {
		if pathContains(root, path) || pathContains(root, target) {
			return false
		}
	}
	return true
}

func (r *executableResolver) lookPath(name string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	if filepath.Base(name) != name || name == "." {
		return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
	}
	for _, dir := range r.dirs {
		base := filepath.Join(dir, name)
		candidates := []string{base}
		if runtime.GOOS == "windows" && len(r.extensions) > 0 {
			if filepath.Ext(name) == "" {
				candidates = nil
			}
			for _, ext := range r.extensions {
				candidates = append(candidates, base+ext)
			}
		}
		for _, candidate := range candidates {
			// Check explicit absolute candidates, never implicit current-directory
			// or PATH searches. Do not accept extensions from a changed PATHEXT.
			found, err := exec.LookPath(candidate)
			if err == nil && found == candidate && r.safe(found) {
				return found, nil
			}
		}
	}
	return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
}

func (r *executableResolver) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	if !filepath.IsAbs(name) {
		var err error
		name, err = r.lookPath(name)
		if err != nil {
			return nil, err
		}
	}
	if !r.safe(name) {
		return nil, &exec.Error{Name: name, Err: exec.ErrNotFound}
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = r.env
	cmd.Dir = r.cwd
	return commandOutput(cmd)
}
