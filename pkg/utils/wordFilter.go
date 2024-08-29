package utils

import (
	"strings"
)

var negativeWords = []string{
	"nigger", "n!gger", "n!gg3r", "nigg3r", // if u add more permutations of the word, ur racist
}

func ContainsNegativeWords(content string) bool {
	lowerContent := strings.ToLower(content)
	for _, word := range negativeWords {
		if strings.Contains(lowerContent, word) {
			return true
		}
	}
	return false
}
