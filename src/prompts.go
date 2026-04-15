package main

import (
	"encoding/json"
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

func LoadToolsFrom(filename string) (*Tools, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	res := &Tools{}
	if err := json.NewDecoder(file).Decode(res); err != nil {
		return nil, err
	}

	return res, nil
}

func GetToolsDefault() *Tools {
	return &Tools{
		Startup: ToolSettings{
			Name:        "get_session_context",
			Description: "CRITICAL: The foundational tool for establishing continuity. To ensure every interaction is personalized and context-aware, this tool should be the **ABSOLUTE FIRST STEP** in any conversation, including greetings. It provides the immediate baseline of the user's current state and recent history.",
		},
		Add: ToolSettings{
			//Name:        "remember",
			//Description: "Stores a new memory. Keep the text short, dense, and precise for best results. You must use this tool immediately and proactively when detecting High-Priority Personal Information (e.g., names, explicit user preferences, personal facts). For Low-Priority data (e.g., names or dates found in quoted text or creative works), only store it if it is directly relevant to the current topic of conversation or if the user explicitly asks you to save that specific piece of information. This ensures the profile remains focused and relevant.",
			Name:        "record_personal_detail",
			Description: "Captures high-value personal information to build a precise user profile. Prioritize density and relevance. Focus on recording information that adds long-term utility, ensuring the memory remains a high-quality asset rather than a cluttered log.",
		},
		Remove: ToolSettings{
			//Name:        "forget",
			//Description: "Deletes an existing memory record by its ID. Use this to free up space when the memory limit is reached.",
			Name:        "prune_memory",
			Description: "Maintains memory integrity and high-fidelity. Use this to remove outdated, incorrect, or redundant information. A clean, accurate profile is more valuable than a large, noisy one; pruning is essential for keeping the user's context sharp and relevant.",
		},
		List: ToolSettings{
			Name:        "query_user_archive",
			Description: "A comprehensive search of the user's long-term history and preferences. If a user's inquiry implies a need for historical depth or specific past details not present in the immediate session context, this tool is the required next step to ensure accuracy and prevent providing uninformed responses.",
		},
		Update: ToolSettings{
			Name:        "update_memory",
			Description: "Updates an existing memory record with new content. Keep the text short, dense, and precise for best results.",
		},
	}
}
