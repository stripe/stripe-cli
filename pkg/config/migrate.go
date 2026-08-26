package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/BurntSushi/toml"

	"github.com/stripe/stripe-cli/pkg/fsutil"
)

// ConfigBackupSuffix is appended to the config file name to name the backup that
// the migration leaves behind, e.g. "config.toml.v1.bak".
const ConfigBackupSuffix = ".v1.bak"

// profileFieldNames are the field names that mark a table as a profile. A
// top-level table is only migrated when it holds at least one of them, so an
// unrecognized table is left where it is instead of being guessed into the
// profiles table. Dual-read means leaving it alone costs nothing: the CLI still
// finds it at the top level.
var profileFieldNames = map[string]bool{
	AccountIDName:               true,
	ColorName:                   true,
	DeviceNameName:              true,
	DisplayNameName:             true,
	IsTermsAcceptanceValidName:  true,
	LiveModeAPIKeyName:          true,
	LiveModeKeyExpiresAtName:    true,
	LiveModePubKeyName:          true,
	SandboxClaimURLName:         true,
	SandboxExpiresAtName:        true,
	TestModeAPIKeyName:          true,
	TestModeKeyExpiresAtName:    true,
	TestModePubKeyName:          true,
	UserIDName:                  true,
	"api_key":                   true,
	"experimental":              true,
	"profile_name":              true,
	"publishable_key":           true,
	"secret_key":                true,
	"test_mode_publishable_key": true,
}

// migrationPlan is the v2 document the migration intends to write: every profile
// under the profiles table, and the CLI's own settings left at the top level.
type migrationPlan struct {
	profiles map[string]map[string]interface{}
	settings map[string]interface{}
}

// MigrateConfigFile rewrites the config file at path in the v2 layout, moving
// every profile under the reserved profiles table and recording
// config_version = 2. It reports whether the file changed.
//
// The original file is copied to path + ConfigBackupSuffix first. The migrated
// document is verified in memory and again after the write; a failure at either
// point restores the backup and returns an error, so a partial or mangled file
// is never left behind.
//
// A file whose config_version is newer than this binary understands is left
// untouched and returns an error.
//
// This function does not decide *whether* to migrate. Callers own that: an
// installed plugin too old to read the v2 layout, or a non-interactive session,
// must not migrate.
func MigrateConfigFile(path string) (bool, error) {
	if err := fsutil.RefuseWriteThroughSymlinkOS(path, filepath.Dir(filepath.Dir(path)), filepath.Base(path)); err != nil {
		return false, err
	}

	original, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Nothing to migrate. A file written from scratch gets its layout from
		// the normal write path.
		return false, nil
	} else if err != nil {
		return false, err
	}

	plan, changed, err := planMigration(original)
	if err != nil {
		return false, err
	}

	if !changed {
		return false, nil
	}

	migrated, err := encodePlan(plan)
	if err != nil {
		return false, err
	}

	// Verify before touching the file at all, so an encoding bug costs the user
	// nothing.
	if err := verifyPlan(plan, migrated); err != nil {
		return false, err
	}

	backupPath := path + ConfigBackupSuffix
	if err := writeFileSync(backupPath, original); err != nil {
		return false, fmt.Errorf("could not back up %s: %w", path, err)
	}

	if err := replaceFileAtomically(path, migrated); err != nil {
		return false, err
	}

	// Re-read what actually landed on disk. A truncated write or a rename that
	// did not take is worth catching here rather than on the next command.
	written, err := os.ReadFile(path)
	if err == nil {
		err = verifyPlan(plan, written)
	}

	if err != nil {
		if restoreErr := writeFileSync(path, original); restoreErr != nil {
			return false, fmt.Errorf("config migration failed (%v) and the original could not be restored: %w; a copy is at %s", err, restoreErr, backupPath)
		}

		return false, fmt.Errorf("config migration failed and was rolled back: %w", err)
	}

	return true, nil
}

// planMigration reads a config document and returns the v2 document to write.
// changed is false when there is nothing to do: an already-migrated file with no
// stray top-level profiles. A config_version newer than this binary understands
// returns an error.
func planMigration(contents []byte) (*migrationPlan, bool, error) {
	// Decode raw TOML rather than going through viper, which lowercases every
	// key. Round-tripping profile names through viper would rename a profile the
	// user created with capitals in it.
	var doc map[string]interface{}
	if err := toml.Unmarshal(contents, &doc); err != nil {
		return nil, false, fmt.Errorf("could not parse config file: %w", err)
	}

	version := configVersionOf(doc)
	if version > MaxSupportedConfigVersion {
		return nil, false, unsupportedConfigVersionError(version)
	}

	plan := &migrationPlan{
		profiles: make(map[string]map[string]interface{}),
		settings: make(map[string]interface{}),
	}

	alreadyV2 := version == ConfigVersionV2

	// Start from the profiles already in the v2 table, if there are any.
	nested, nestedIsContainer := profilesContainer(doc, alreadyV2)
	if nestedIsContainer {
		for name, value := range nested {
			if table, ok := toStringMap(value); ok {
				plan.profiles[name] = table
			}
		}
	}

	moved := 0

	for key, value := range doc {
		if key == ConfigVersionName {
			continue
		}

		if key == ProfilesTableName && nestedIsContainer {
			continue
		}

		table, isTable := toStringMap(value)
		if !isTable || reservedProfileNameException(key, nestedIsContainer) || !looksLikeProfile(table) {
			plan.settings[key] = value
			continue
		}

		// A field already present in the v2 table wins: it is the copy the
		// current CLI reads and writes, so the top-level table is the stale one.
		if existing, ok := plan.profiles[key]; ok {
			for field, fieldValue := range table {
				if _, taken := existing[field]; !taken {
					existing[field] = fieldValue
				}
			}
		} else {
			plan.profiles[key] = table
		}

		moved++
	}

	return plan, moved > 0 || !alreadyV2, nil
}

// reservedProfileNameException reports whether a top-level key must be treated
// as a setting rather than as a profile. Only the profiles table itself
// qualifies, and only when it is acting as the v2 container.
func reservedProfileNameException(key string, nestedIsContainer bool) bool {
	if key == ProfilesTableName {
		return nestedIsContainer
	}

	return reservedTopLevelKeys[key]
}

// profilesContainer returns the v2 profiles table, if the document has one.
//
// A v1 file can contain a profile literally named "profiles", which occupies the
// same key as the v2 container: nothing stops `stripe login --project-name
// profiles`. The two are distinguishable because a profile holds scalar fields
// while the container holds only tables, and because a migrated file says so
// with config_version.
func profilesContainer(doc map[string]interface{}, alreadyV2 bool) (map[string]interface{}, bool) {
	table, ok := toStringMap(doc[ProfilesTableName])
	if !ok {
		return nil, false
	}

	if alreadyV2 {
		return table, true
	}

	// In a v1 file, treat it as the container only if it cannot be a profile.
	if looksLikeProfile(table) {
		return nil, false
	}

	for _, value := range table {
		if _, isTable := toStringMap(value); !isTable {
			return nil, false
		}
	}

	return table, true
}

// looksLikeProfile reports whether a table holds at least one recognized profile
// field.
func looksLikeProfile(table map[string]interface{}) bool {
	for field := range table {
		if profileFieldNames[field] {
			return true
		}
	}

	return false
}

// configVersionOf reads config_version out of a raw TOML document. TOML integers
// decode as int64.
func configVersionOf(doc map[string]interface{}) int {
	switch version := doc[ConfigVersionName].(type) {
	case int64:
		return int(version)
	case int:
		return version
	}

	return 0
}

// encodePlan serializes a plan as a v2 TOML document.
func encodePlan(plan *migrationPlan) ([]byte, error) {
	doc := make(map[string]interface{}, len(plan.settings)+2)
	for key, value := range plan.settings {
		doc[key] = value
	}

	doc[ConfigVersionName] = ConfigVersionV2

	profiles := make(map[string]interface{}, len(plan.profiles))
	for name, table := range plan.profiles {
		profiles[name] = table
	}
	doc[ProfilesTableName] = profiles

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(doc); err != nil {
		return nil, fmt.Errorf("could not encode migrated config: %w", err)
	}

	return buf.Bytes(), nil
}

// verifyPlan re-parses an encoded document and checks that it says exactly what
// the plan intended: every profile field under the profiles table with its
// original value, every setting still at the top level, and nothing extra.
//
// This is what makes the migration safe to run unattended. A serialization bug
// that folded a profile name's case or dropped a field would otherwise silently
// cost the user a credential.
func verifyPlan(plan *migrationPlan, encoded []byte) error {
	var doc map[string]interface{}
	if err := toml.Unmarshal(encoded, &doc); err != nil {
		return fmt.Errorf("migrated config does not parse: %w", err)
	}

	if got := configVersionOf(doc); got != ConfigVersionV2 {
		return fmt.Errorf("migrated config has config_version %d, want %d", got, ConfigVersionV2)
	}

	profiles, ok := toStringMap(doc[ProfilesTableName])
	if !ok {
		return fmt.Errorf("migrated config has no %s table", ProfilesTableName)
	}

	for _, name := range sortedProfileNames(plan.profiles) {
		table, ok := toStringMap(profiles[name])
		if !ok {
			return fmt.Errorf("profile %q is missing from the migrated config", name)
		}

		for _, field := range sortedKeys(plan.profiles[name]) {
			want := plan.profiles[name][field]
			got, present := table[field]
			if !present {
				return fmt.Errorf("profile %q lost field %q in the migrated config", name, field)
			}

			if !reflect.DeepEqual(want, got) {
				return fmt.Errorf("profile %q field %q changed value in the migrated config", name, field)
			}
		}

		if len(table) != len(plan.profiles[name]) {
			return fmt.Errorf("profile %q gained fields in the migrated config", name)
		}
	}

	if len(profiles) != len(plan.profiles) {
		return fmt.Errorf("migrated config has %d profiles, want %d", len(profiles), len(plan.profiles))
	}

	for _, key := range sortedKeys(plan.settings) {
		got, present := doc[key]
		if !present {
			return fmt.Errorf("setting %q is missing from the migrated config", key)
		}

		if !reflect.DeepEqual(plan.settings[key], got) {
			return fmt.Errorf("setting %q changed value in the migrated config", key)
		}
	}

	// The settings, plus config_version and the profiles table.
	if len(doc) != len(plan.settings)+2 {
		return fmt.Errorf("migrated config has %d top-level keys, want %d", len(doc), len(plan.settings)+2)
	}

	return nil
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}

func sortedProfileNames(m map[string]map[string]interface{}) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// replaceFileAtomically writes contents to a temp file in the same directory and
// renames it over path, so a reader either sees the old file or the new one.
func replaceFileAtomically(path string, contents []byte) error {
	dir := filepath.Dir(path)

	temp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()

	// Best effort: a failure below leaves the temp file behind, but the config
	// file itself is untouched.
	defer os.Remove(tempPath)

	if err := writeAndSync(temp, contents); err != nil {
		temp.Close()
		return err
	}

	if err := temp.Chmod(os.FileMode(0600)); err != nil {
		temp.Close()
		return err
	}

	if err := temp.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}

// writeFileSync writes contents to path with the config file's permissions and
// flushes them to disk before returning.
func writeFileSync(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(0600))
	if err != nil {
		return err
	}

	if err := writeAndSync(file, contents); err != nil {
		file.Close()
		return err
	}

	return file.Close()
}

func writeAndSync(file *os.File, contents []byte) error {
	if _, err := file.Write(contents); err != nil {
		return err
	}

	return file.Sync()
}
