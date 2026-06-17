package config

// MemoryStore is an in-memory SecureStore for use in tests.
type MemoryStore struct {
	items map[string][]byte
}

// NewMemoryStore creates a MemoryStore optionally pre-populated with data.
func NewMemoryStore(initial map[string][]byte) *MemoryStore {
	m := &MemoryStore{items: make(map[string][]byte)}
	for k, v := range initial {
		m.items[k] = v
	}
	return m
}

func (m *MemoryStore) Get(key string) ([]byte, error) {
	data, ok := m.items[key]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return data, nil
}

func (m *MemoryStore) Set(key string, data []byte, description string) error {
	m.items[key] = data
	return nil
}

func (m *MemoryStore) Remove(key string) error {
	if _, ok := m.items[key]; !ok {
		return ErrKeyNotFound
	}
	delete(m.items, key)
	return nil
}

