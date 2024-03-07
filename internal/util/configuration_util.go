package util

import "fmt"

// MakeConfigFileContent returns the content of a configuration file
func MakeConfigFileContent(config map[string]string) string {
	content := ""
	if len(config) == 0 {
		return content
	}
	for k, v := range config {
		content += fmt.Sprintf("%s %s\n", k, v)
	}
	return content
}

func MakePropertiesFileContent(config map[string]string) string {
	content := ""
	if len(config) == 0 {
		return content
	}
	for k, v := range config {
		content += fmt.Sprintf("%s=%s\n", k, v)
	}
	return content
}
