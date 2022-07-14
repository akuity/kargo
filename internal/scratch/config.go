package scratch

import (
	"encoding/json"
	"io/ioutil"

	"github.com/akuityio/k8sta/internal/common/file"
	"github.com/akuityio/k8sta/internal/common/os"
	"github.com/pkg/errors"
)

// Config is the K8sTA configuration object.
type Config interface {
	AddLine(Line)
	LineCount() int
	GetLineByName(string) (Line, bool)
	GetLineByImageRepository(string) (Line, bool)
}

type config struct {
	linesByName      map[string]Line
	linesByImageRepo map[string]Line
}

func NewConfig() Config {
	return &config{
		linesByName:      map[string]Line{},
		linesByImageRepo: map[string]Line{},
	}
}

func (c *config) AddLine(line Line) {
	c.linesByName[line.Name] = line
	c.linesByImageRepo[line.ImageRepository] = line
}

func (c *config) LineCount() int {
	return len(c.linesByName)
}

func (c *config) GetLineByName(name string) (Line, bool) {
	line, ok := c.linesByName[name]
	return line, ok
}

func (c *config) GetLineByImageRepository(repo string) (Line, bool) {
	line, ok := c.linesByImageRepo[repo]
	return line, ok
}

func K8staConfig() (Config, error) {
	config := NewConfig()
	configPath, err := os.GetRequiredEnvVar("CONFIG_PATH")
	if err != nil {
		return config, err
	}
	var exists bool
	if exists, err = file.Exists(configPath); err != nil {
		return config, err
	}
	if !exists {
		return config, errors.Errorf("file %s does not exist", configPath)
	}
	linesBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}
	lines := []Line{}
	if err := json.Unmarshal(linesBytes, &lines); err != nil {
		return config, err
	}
	for _, line := range lines {
		config.AddLine(line)
	}
	return config, nil
}
