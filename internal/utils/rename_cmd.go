package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
)

// BuildCommandReplaceMapping reads the config file with the command-replace lines and build a mapping of
// bad lines are ignored silently
func BuildCommandReplaceMapping(filePath string, log logr.Logger) map[string]string {
	mapping := make(map[string]string)
	log.Info("Building Command Replace Mapping", "FilePath", filePath)
	if filePath == "" {
		return mapping
	}
	log.Info("Building Command Replace Mapping", "FilePath Not EMPTY", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		log.Error(err, fmt.Sprintf("cannot open %s", filePath))
		return mapping
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		elems := strings.Fields(scanner.Text())
		if len(elems) == 3 && strings.ToLower(elems[0]) == "rename-command" {
			mapping[strings.ToUpper(elems[1])] = elems[2]
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error(err, fmt.Sprintf("cannot parse %s", filePath))
		return mapping
	}
	return mapping
}
