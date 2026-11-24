package agentsetup

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// executableResolver excludes project-controlled locations, not arbitrary
// untrusted software. Non-project installation directories must be trusted.
// Like ordinary executable lookup, this does not prevent replacement races.
type executableResolver struct {
	dirs       []string
	roots      []string
	rootInfos  []os.FileInfo
	extensions []string
	env        []string
	cwd        string
	err        error
}

// lazyExecutableResolver defers working-directory and PATH probing until the
// first lookup, so commands unrelated to agent setup never touch PATH entries
// such as unavailable network mounts.
func lazyExecutableResolver() (func(string) (string, error), RunCommandFunc) {
	get := sync.OnceValue(newExecutableResolver)
	lookPath := func(name string) (string, error) { return get().lookPath(name) }
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return get().run(ctx, name, args...)
	}
	return lookPath, run
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
	candidates, err := projectRootCandidates(slices.Compact([]string{r.cwd, resolved}))
	if err != nil {
		r.err = err
		return r
	}
	// The home directory and its ancestors hold user-level installations such
	// as ~/.local/bin, so they are never treated as project roots, even when
	// they contain a marker such as a dotfiles repository.
	// Check both spellings: a symlinked home has different lexical ancestors.
	var homes []string
	if home, err := os.UserHomeDir(); err == nil && comparableAbsolutePath(home) {
		homes = append(homes, home)
		if resolved, err := filepath.EvalSymlinks(home); err == nil && resolved != home {
			homes = append(homes, resolved)
		}
	}
candidates:
	for _, root := range candidates {
		info, err := os.Stat(root)
		if err != nil {
			r.err = err
			return r
		}
		for _, home := range homes {
			containsHome, err := underAny(home, []os.FileInfo{info})
			if err != nil {
				r.err = err
				return r
			}
			if containsHome {
				continue candidates
			}
		}
		r.roots = append(r.roots, root)
		r.rootInfos = append(r.rootInfos, info)
	}
	var path, pathExt string
	filtered := make([]string, 0, len(r.env)+1)
	for _, entry := range r.env {
		key, value, _ := strings.Cut(entry, "=")
		if key == "PATH" || runtime.GOOS == "windows" && strings.EqualFold(key, "PATH") {
			path = value
			continue
		}
		filtered = append(filtered, entry)
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
		// Validate the exported spelling: cleaning "link/.." can change the
		// directory it names.
		if dir = filepath.Clean(dir); r.safe(dir) {
			r.dirs = append(r.dirs, dir)
		}
	}
	filtered = append(filtered, "PATH="+joinPathList(r.dirs))
	r.env = filtered
	return r
}

// projectRootCandidates returns each start directory and every ancestor that
// contains a project marker, along with the symlink targets of those ancestors.
func projectRootCandidates(starts []string) ([]string, error) {
	var candidates []string
	for _, start := range starts {
		if !comparableAbsolutePath(start) {
			return nil, &exec.Error{Name: start, Err: exec.ErrNotFound}
		}
		candidates = append(candidates, start)
		for dir := start; ; dir = filepath.Dir(dir) {
			for _, marker := range []string{".git", ".hg", "package.json", "go.mod", "Cargo.toml", "pyproject.toml"} {
				if _, err := os.Lstat(filepath.Join(dir, marker)); err == nil {
					candidates = append(candidates, dir)
					target, err := filepath.EvalSymlinks(dir)
					if err != nil {
						return nil, err
					}
					if target != dir {
						candidates = append(candidates, target)
					}
					break
				} else if !os.IsNotExist(err) {
					return nil, err
				}
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	return candidates, nil
}

// joinPathList reverses filepath.SplitList. On Windows, SplitList unquotes
// entries containing ";", so they must be quoted again to stay one entry.
func joinPathList(dirs []string) string {
	quoted := make([]string, len(dirs))
	for i, dir := range dirs {
		if strings.Contains(dir, string(os.PathListSeparator)) {
			dir = `"` + dir + `"`
		}
		quoted[i] = dir
	}
	return strings.Join(quoted, string(os.PathListSeparator))
}

func pathContains(root, path string) bool {
	if runtime.GOOS == "windows" {
		root, path = windowsComparisonPath(root), windowsComparisonPath(path)
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// underAny reports whether path or one of its ancestors is the same file as
// any of roots. Unlike a lexical comparison, this recognizes alternate
// spellings such as case aliases on case-insensitive filesystems.
func underAny(path string, roots []os.FileInfo) (bool, error) {
	for dir := filepath.Clean(path); ; dir = filepath.Dir(dir) {
		info, err := os.Stat(dir)
		if err == nil {
			for _, root := range roots {
				if os.SameFile(info, root) {
					return true, nil
				}
			}
		} else if !os.IsNotExist(err) {
			return false, err
		}
		if filepath.Dir(dir) == dir {
			return false, nil
		}
	}
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
		path = windowsComparisonPath(path)
		return !strings.HasPrefix(path, `\\.\`) && filepath.IsAbs(path)
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
	for _, p := range []string{path, target} {
		if under, err := underAny(p, r.rootInfos); err != nil || under {
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
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	// Prefer stderr for a process that started, otherwise use the output.
	stderr := out
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr = exitErr.Stderr
	}
	if msg := errorFromOutput(stderr); msg != "" {
		return out, errorcategory.New(errorcategory.Internal, msg)
	}
	return out, err
}
