package doctor

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var dependencyName = regexp.MustCompile(`^[@a-zA-Z0-9][a-zA-Z0-9@./_-]{0,159}$`)
var dependencyVersion = regexp.MustCompile(`^[a-zA-Z0-9.*^~+<>=!_-]{1,80}$`)

func manifestDependencies(path string, data []byte) []Dependency {
	out := []Dependency{}
	add := func(name, version string) {
		if len(out) >= 64 || !dependencyName.MatchString(name) {
			return
		}
		if !dependencyVersion.MatchString(version) {
			version = "unknown"
		}
		out = append(out, Dependency{Manifest: path, Name: name, Version: version, Resolution: "external or local replacement; implementation not followed"})
	}
	switch filepath.Base(path) {
	case "go.mod":
		requireGroup := false
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(strings.SplitN(line, "//", 2)[0])
			if len(fields) == 0 {
				continue
			}
			if fields[0] == ")" {
				requireGroup = false
				continue
			}
			if fields[0] == "require" {
				fields = fields[1:]
				if len(fields) == 1 && fields[0] == "(" {
					requireGroup = true
					continue
				}
				if len(fields) >= 2 {
					add(fields[0], fields[1])
				}
				continue
			}
			if requireGroup && len(fields) >= 2 {
				add(fields[0], fields[1])
			}
		}
	case "package.json", "composer.json":
		var manifest map[string]json.RawMessage
		if json.Unmarshal(data, &manifest) != nil {
			return out
		}
		for _, key := range []string{"dependencies", "devDependencies", "peerDependencies", "require", "require-dev"} {
			var deps map[string]string
			if json.Unmarshal(manifest[key], &deps) != nil {
				continue
			}
			names := []string{}
			for name := range deps {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				add(name, deps[name])
			}
		}
	}
	return out
}
