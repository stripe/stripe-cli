package doctor

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var apiVersionPattern = regexp.MustCompile(`^20[0-9]{2}-[0-9]{2}-[0-9]{2}(\.[a-z]+)?$`)

func LoadHandoff(path, root string, discovery Discovery) ([]SourceFlow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("investigation file unavailable")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxInput+1))
	if err != nil {
		return nil, errors.New("investigation file unreadable")
	}
	var h Handoff
	if err := strictJSON(data, &h); err != nil {
		return nil, err
	}
	if h.Snapshot != discovery.Snapshot || len(h.Flows) == 0 || len(h.Flows) > 128 {
		return nil, errors.New("stale or unbounded investigation")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("source root unavailable")
	}
	defer r.Close()
	ids := map[string]bool{}
	for i := range h.Flows {
		flow := &h.Flows[i]
		if flow.ID == "" || len(flow.ID) > 80 || ids[flow.ID] || flow.Snapshot != h.Snapshot || (flow.Method != "POST" && flow.Method != "unknown") || !supportedPath(flow.Path) || len(flow.Callers) > 32 || len(flow.Builders) > 32 || len(flow.Unresolved) > 32 {
			return nil, errors.New("invalid flow evidence")
		}
		ids[flow.ID] = true
		if flow.Context.Account != "unknown" && flow.Context.Account != "profile" && flow.Context.Account != "connect" {
			return nil, errors.New("invalid account context")
		}
		if flow.Context.Mode != "unknown" && flow.Context.Mode != "test" && flow.Context.Mode != "live" {
			return nil, errors.New("invalid mode context")
		}
		if flow.Context.APIVersion != "" && !apiVersionPattern.MatchString(flow.Context.APIVersion) {
			return nil, errors.New("invalid per-flow API version")
		}
		if len(flow.Context.Configuration) > 128 {
			return nil, errors.New("configuration identifier exceeds budget")
		}
		switch flow.Automatic {
		case "enabled", "disabled", "absent", "unknown":
		default:
			return nil, errors.New("invalid automatic payment methods evidence")
		}
		switch flow.Pinned.Status {
		case "known":
			if len(flow.Pinned.Alternatives) > 0 {
				return nil, errors.New("known methods cannot have alternatives")
			}
		case "conditional":
			if len(flow.Pinned.Alternatives) == 0 || len(flow.Pinned.Values) > 0 {
				return nil, errors.New("conditional methods require separate alternatives")
			}
		case "absent", "unknown":
			if len(flow.Pinned.Values) > 0 || len(flow.Pinned.Alternatives) > 0 {
				return nil, errors.New("absent/unknown methods cannot have values")
			}
		default:
			return nil, errors.New("invalid pinned method status")
		}
		sets := append([][]string{flow.Pinned.Values}, flow.Pinned.Alternatives...)
		if len(sets) > 17 {
			return nil, errors.New("method alternatives exceed budget")
		}
		for _, set := range sets {
			if len(set) > 64 {
				return nil, errors.New("methods exceed budget")
			}
			seen := map[string]bool{}
			for _, v := range set {
				if !identifier.MatchString(v) || seen[v] {
					return nil, errors.New("invalid method evidence")
				}
				seen[v] = true
			}
		}
		citations := append(append([]Citation{flow.Boundary}, flow.Callers...), flow.Builders...)
		for _, c := range citations {
			if err := validateCitation(r, c); err != nil {
				return nil, err
			}
		}
		// Agent assertions remain reviewable claims even when citations are current.
		flow.Unresolved = append(flow.Unresolved, "Agent evidence requires human semantic review; citations prove provenance, not inference")
	}
	return h.Flows, nil
}

func validateCitation(root *os.Root, c Citation) error {
	if !filepath.IsLocal(c.File) || filepath.ToSlash(filepath.Clean(c.File)) != c.File || strings.Contains(c.File, "\\") || c.Start < 1 || c.End < c.Start || c.End-c.Start > 500 || c.Symbol == "" || len(c.Symbol) > 128 {
		return errors.New("invalid source citation")
	}
	parts := strings.Split(c.File, "/")
	for i := range parts {
		p := filepath.Join(parts[:i+1]...)
		info, err := root.Lstat(p)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("missing or symlink source citation")
		}
		if i < len(parts)-1 && excludedDir(parts[i]) {
			return errors.New("citation points to excluded source")
		}
	}
	f, err := root.Open(c.File)
	if err != nil {
		return errors.New("missing source citation")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxInput+1))
	if err != nil || len(data) > maxInput || digest(data) != c.SHA256 || c.End > strings.Count(string(data), "\n")+1 {
		return errors.New("stale or invalid source citation")
	}
	return nil
}
