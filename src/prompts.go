package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type ToolSettings struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Tools struct {
	ChatSessionStartup ToolSettings `json:"startup"`
	Remember           ToolSettings `json:"remember"`
	Forget             ToolSettings `json:"forget"`
	ListMemories       ToolSettings `json:"list"`
	UpdateMemory       ToolSettings `json:"update"`
}

func LoadToolsFrom(filename string) (*Tools, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Unable to load tool definition file '%s': %v", filename, err)
	}

	res := &Tools{}
	if err := json.Unmarshal(data, res); err != nil {
		return nil, fmt.Errorf("Unable to parse tool definitions: %v", err)
	}

	return res, nil
}

func GetToolsDefault() *Tools {
	return &Tools{
		ChatSessionStartup: ToolSettings{
			Name:        "get_session_context",
			Description: "The foundational tool for establishing continuity. To ensure every interaction is personalized and context-aware, this tool should be the first step in any conversation, including greetings. It provides the immediate baseline of the user's current state and recent history.",
		},
		Remember: ToolSettings{
			//Name:        "remember",
			//Description: "Stores a new memory. Keep the text short, dense, and precise for best results. You must use this tool immediately and proactively when detecting High-Priority Personal Information (e.g., names, explicit user preferences, personal facts). For Low-Priority data (e.g., names or dates found in quoted text or creative works), only store it if it is directly relevant to the current topic of conversation or if the user explicitly asks you to save that specific piece of information. This ensures the profile remains focused and relevant.",
			Name:        "record_personal_detail",
			Description: "Captures high-value personal information to build a precise user profile. Prioritize density and relevance. Focus on recording information that adds long-term utility, ensuring the memory remains a high-quality asset rather than a cluttered log.",
		},
		Forget: ToolSettings{
			//Name:        "forget",
			//Description: "Deletes an existing memory record by its ID. Use this to free up space when the memory limit is reached.",
			Name:        "prune_memory",
			Description: "Maintains memory integrity and high-fidelity. Use this to remove outdated, incorrect, or redundant information. A clean, accurate profile is more valuable than a large, noisy one; pruning is essential for keeping the user's context sharp and relevant.",
		},
		ListMemories: ToolSettings{
			Name:        "query_user_archive",
			Description: "A comprehensive search of the user's long-term history and preferences. If a user's inquiry implies a need for historical depth or specific past details not present in the immediate session context, this tool is the required next step to ensure accuracy and prevent providing uninformed responses.",
		},
		UpdateMemory: ToolSettings{
			Name:        "update_memory",
			Description: "Updates an existing memory record with new content. Keep the text short, dense, and precise for best results.",
		},
	}
}
