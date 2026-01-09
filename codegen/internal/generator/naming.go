package generator

import (
	"strings"
	"unicode"
)

// pascalCase converts arbitrary strings (optionally adding a suffix) into valid
// PascalCase identifiers.
func pascalCase(input string, suffix string) string {
	words := tokenize(input)
	if len(words) == 0 {
		words = []string{"Value"}
	}
	for i, word := range words {
		words[i] = capitalize(word)
	}
	name := strings.Join(words, "")
	if name == "" {
		name = "Value"
	}
	if suffix != "" {
		name += suffix
	}
	if len(name) > 0 && unicode.IsDigit([]rune(name)[0]) {
		name = "Api" + name
	}
	return name
}

// camelCase builds lowerCamelCase identifiers, falling back to sane defaults
// for empty inputs.
func camelCase(input string, fallback string) string {
	words := tokenize(input)
	if len(words) == 0 {
		if fallback != "" {
			return fallback
		}
		return "value"
	}
	for i, word := range words {
		if i == 0 {
			words[i] = strings.ToLower(word)
		} else {
			words[i] = capitalize(word)
		}
	}
	name := strings.Join(words, "")
	if len(name) > 0 && unicode.IsDigit([]rune(name)[0]) {
		name = "value" + name
	}
	return name
}

// enumConstantName converts an enum value into a valid Java enum constant name.
func enumConstantName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "EMPTY"
	}
	words := tokenize(trimmed)
	if len(words) == 0 {
		return "VALUE"
	}
	for i, word := range words {
		words[i] = strings.ToUpper(word)
	}
	name := strings.Join(words, "_")
	if name == "" {
		name = "VALUE"
	}
	if len(name) > 0 && unicode.IsDigit([]rune(name)[0]) {
		name = "VALUE_" + name
	}
	return name
}

// asyncClientClassName returns the asynchronous variant of a generated client.
func asyncClientClassName(name string) string {
	if strings.HasSuffix(name, "Client") {
		return strings.TrimSuffix(name, "Client") + "AsyncClient"
	}
	return name + "AsyncClient"
}

// tokenize splits identifiers or sentences into constituent alphanumeric words
// that can later be re-cased.
func tokenize(input string) []string {
	if input == "" {
		return nil
	}
	var (
		words   []string
		current []rune
		prev    rune
	)
	flush := func() {
		if len(current) == 0 {
			return
		}
		words = append(words, string(current))
		current = current[:0]
	}
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if len(current) > 0 {
				prevIsLower := unicode.IsLower(prev)
				currIsUpper := unicode.IsUpper(r)
				currIsDigit := unicode.IsDigit(r)
				prevIsDigit := unicode.IsDigit(prev)
				if (prevIsLower && currIsUpper) || (prevIsDigit && !currIsDigit) {
					flush()
				}
			}
			current = append(current, r)
			prev = r
		} else {
			flush()
			prev = 0
		}
	}
	flush()
	return words
}

// capitalize lowercases a string and then uppercases the first rune.
func capitalize(word string) string {
	if word == "" {
		return ""
	}
	runes := []rune(strings.ToLower(word))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
