package composefile

// ConfigSpec is a top-level configs: entry (Compose v3 subset for Podbay import).
type ConfigSpec struct {
	File     string `yaml:"file,omitempty"`
	External bool   `yaml:"external,omitempty"`
}

// SecretSpec is a top-level secrets: entry (Compose v3 subset for Podbay import).
type SecretSpec struct {
	File     string `yaml:"file,omitempty"`
	External bool   `yaml:"external,omitempty"`
}
