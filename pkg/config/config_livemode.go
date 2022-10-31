//go:build !arm64
// +build !arm64

package config

func deleteLivemodeKey(key string, profile string) error {
	fieldID := profile + "." + key
	existingKeys, err := KeyRing.Keys()
	if err != nil {
		return err
	}
	for _, item := range existingKeys {
		if item == fieldID {
			KeyRing.Remove(fieldID)
			return nil
		}
	}
	return nil
}
