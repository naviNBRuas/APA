package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifest_Defaults(t *testing.T) {
	m := &Manifest{}
	assert.Empty(t, m.Name)
	assert.Empty(t, m.Version)
	assert.Empty(t, m.Path)
	assert.Empty(t, m.Hash)
	assert.Nil(t, m.Capabilities)
	assert.Empty(t, m.Policy)
}

func TestManifest_Full(t *testing.T) {
	m := &Manifest{
		Name:         "test-controller",
		Version:      "1.0.0",
		Path:         "/usr/bin/test",
		Hash:         "abc123",
		Capabilities: []string{"network", "storage"},
		Policy:       "allow_all",
	}
	assert.Equal(t, "test-controller", m.Name)
	assert.Equal(t, "1.0.0", m.Version)
	assert.Equal(t, "/usr/bin/test", m.Path)
	assert.Equal(t, "abc123", m.Hash)
	assert.Equal(t, []string{"network", "storage"}, m.Capabilities)
	assert.Equal(t, "allow_all", m.Policy)
}
