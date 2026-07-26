package main

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v4"
)

type ToolDefinition struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type ToolDefinitions struct {
	Startup ToolDefinition `yaml:"startup"`
	Add     ToolDefinition `yaml:"remember"`
	Remove  ToolDefinition `yaml:"forget"`
	List    ToolDefinition `yaml:"list"`
	Update  ToolDefinition `yaml:"update"`
}

func loadTools(reader io.Reader) (*ToolDefinitions, error) {
	res := &ToolDefinitions{}
	if err := yaml.NewDecoder(reader).Decode(res); err != nil {
		return nil, err
	}
	return res, nil
}

func LoadToolsFromFile(filename string) (*ToolDefinitions, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return loadTools(file)
}

//go:embed assets
var embeddedAssets embed.FS

func GetToolsDefault() *ToolDefinitions {
	fileData, err := embeddedAssets.ReadFile("assets/tools-default.yml")
	if err != nil {
		panic(fmt.Errorf("Unable to read default tool definitions: %v", err))
	}
	tools, err := loadTools(bytes.NewReader(fileData))
	if err != nil {
		panic(fmt.Errorf("Unable to parse default tool definitions: %v", err))
	}

	return tools
}
