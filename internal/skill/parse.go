package skill

import "gopkg.in/yaml.v3"

func parseManifest(data []byte, m *Manifest) error {
	return yaml.Unmarshal(data, m)
}
