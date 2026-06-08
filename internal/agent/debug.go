package agent

import (
	"fmt"
	"os"
	"strings"
)

func debugEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AGENT_DEBUG"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func debugLog(format string, args ...any) {
	if debugEnabled() {
		fmt.Printf("[agent] "+format+"\n", args...)
	}
}
