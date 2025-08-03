package queueoutputcreator

import "encoding/json"

type Metadata struct {
	metadata map[string]string
}

func DeserializeMetadata(data []byte) *Metadata {
	if data == nil {
		return &Metadata{}
	}

	var metadata map[string]string
	err := json.Unmarshal(data, &metadata)
	if err != nil {
		return &Metadata{}
	}

	return &Metadata{metadata: metadata}
}

func (m Metadata) Get(key string) (string, bool) {
	if m.metadata == nil {
		return "", false
	}

	value, exists := m.metadata[key]
	return value, exists
}

func (m *Metadata) Set(key, value string) {
	if m.metadata == nil {
		m.metadata = make(map[string]string)
	}
	m.metadata[key] = value
}

func (m *Metadata) IsEmpty() bool {
	return len(m.metadata) == 0
}

func (m *Metadata) Serialize() []byte {
	if m.metadata == nil {
		return nil
	}

	jsonData, err := json.Marshal(m.metadata)
	if err != nil {
		return nil
	}
	return jsonData
}

func (m *Metadata) ToMap() map[string]string {
	if m.metadata == nil {
		return make(map[string]string)
	}
	return m.metadata
}
