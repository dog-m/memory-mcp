package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type ToolSettings struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Tools struct {
	Startup ToolSettings `json:"startup"`
	Add     ToolSettings `json:"remember"`
	Remove  ToolSettings `json:"forget"`
	List    ToolSettings `json:"list"`
	Update  ToolSettings `json:"update"`
}

func loadTools(reader io.Reader) (*Tools, error) {
	res := &Tools{}
	if err := json.NewDecoder(reader).Decode(res); err != nil {
		return nil, err
	}
	return res, nil
}

func LoadToolsFromFile(filename string) (*Tools, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return loadTools(file)
}

//go:embed assets
var embeddedAssets embed.FS

func GetToolsDefault() *Tools {
	fileData, err := embeddedAssets.ReadFile("assets/tools-default.json")
	if err != nil {
		panic(fmt.Errorf("Unable to read default tool definitions: %v", err))
	}
	tools, err := loadTools(bytes.NewReader(fileData))
	if err != nil {
		panic(fmt.Errorf("Unable to parse default tool definitions: %v", err))
	}

	return tools
}
