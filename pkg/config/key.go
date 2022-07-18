package config

import (
	"fmt"

	"github.com/99designs/keyring"
)

// StoreKey
func StoreKey() {
	ring, _ := keyring.Open(keyring.Config{
		ServiceName: "example",
	})

	_ = ring.Set(keyring.Item{
		Key:         "foo",
		Data:        []byte("secret-bar"),
		Description: "im description",
		Label:       "im label",
	})

	i, _ := ring.Get("foo")
	fmt.Println(i)
	fmt.Println(ring.Keys())
}
