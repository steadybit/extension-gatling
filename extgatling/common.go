package extgatling

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"strings"
)

const (
	actionId   = "com.github.steadybit.extension_gatling.run"
	actionIcon = "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjUiIHZpZXdCb3g9IjAgMCAyNCAyNSIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cGF0aCBkPSJNMTcuMDA4IDIuNUg2Ljk5MkM0LjI0OCAyLjUgMiA0Ljc0OCAyIDcuNDkydjEwLjAxNkMyIDIwLjI1MiA0LjI0OCAyMi41IDYuOTkyIDIyLjVoMTAuMDE2YzIuNzQ0IDAgNC45OTItMi4yNDggNC45OTItNC45OTJWNy40OTJDMjIgNC43NDggMTkuNzUyIDIuNSAxNy4wMDggMi41ek0xMi44MSAxOS4xNjFjLTMuNzAzLjQ5Ni03LjMwNi0yLjEzMi03LjcwMy01LjYzNi0uNDYyLTQuMSAyLjIxNS03LjQwNSA2LjMxNC03LjY3IDEuNzg2LS4xMTUgMy42MDQtLjA2NiA1LjQwNSAwIC4yOTguMDE3IDEuMzU2LjAxNyAxLjM1Ni4wMTd2LjUxMmMuMDMzLjU0Ni0uMTQ5IDEuMzIzLS41NDYgMS42MzctLjYxMS40NzktMS40MzguNzc3LTIuMjE0Ljg5Mi0xLjAyNS4xNDktMi4wNjcuMDE3LTMuMTA4LjA1LTIuMjk3LjA1LTMuNzY5IDEuMDU4LTQuMDY2IDIuNzkzLS4yOTggMS43NTIuMjQ4IDMuMTU3IDEuOTE3IDMuOTUgMS44MzUuODc3IDMuNzAzLjM2NCA1LjM0LTEuNjM2LTIuMDY3LS4zOTYtNC43MTIuNjQ1LTQuOTkzLTIuODI2aDguMzMxYy42NzggNC4xMzItMS44MDIgNy4zMzktNi4wMzMgNy45MTd6IiBmaWxsPSJjdXJyZW50Q29sb3IiLz48L3N2Zz4="
)

func stdOutToLog(lines []string) {
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.ReplaceAll(line, "\n", ""))
		if len(trimmed) > 0 {
			log.Info().Msgf("---- %s", trimmed)
		}
	}
}

func stdOutToMessages(lines []string) []action_kit_api.Message {
	var messages []action_kit_api.Message
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.ReplaceAll(line, "\n", ""))
		if len(trimmed) > 0 {
			messages = append(messages, action_kit_api.Message{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: trimmed,
			})
		}
	}
	return messages
}
