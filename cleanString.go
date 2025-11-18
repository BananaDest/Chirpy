package main

import "strings"

func CleanString(str string) string {
	words := strings.Split(str, " ")
	for i, word := range words {
		switch strings.ToLower(word) {
		case "kerfuffle":
			words[i] = "****"
		case "sharbert":
			words[i] = "****"
		case "fornax":
			words[i] = "****"
		}
	}
	result := strings.Join(words, " ")
	return result
}
