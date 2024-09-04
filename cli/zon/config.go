package zon

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type Config struct {
	Name              string                `json,zon:"name"`
	Version           string                `json,zon:"version"`
	MinimumZigVersion string                `json,zon:"minimum_zig_version"`
	Dependencies      map[string]Dependency `json,zon:"dependencies"`
	Paths             []string              `json,zon:"paths"`
}

type Dependency struct {
	URL  string
	Hash string
	Path string
	Lazy bool
}

func ParseConfig(reader io.Reader) (*Config, error) {
	scanner := bufio.NewScanner(reader)
	config := &Config{
		Dependencies: make(map[string]Dependency),
	}

	tokens := tokenize(scanner)
	fmt.Println(tokens)

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

// tokenize tokenizes a scanned zon file
func tokenize(scanner *bufio.Scanner) []tc {

	tokens := make([]tc, 0)
	depth := 0
	cursor := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "//") {
			cursor += len(line)
			continue
		}

		for _, char := range line {
			switch char {
			case '.':
				tokens = append(tokens, tc{lexeme: ".", token: tkPeriod, depth: depth})
			case '{':
				depth++
				tokens = append(tokens, tc{lexeme: "{", token: tkLeftBrace, depth: depth})
            case '}':
				tokens = append(tokens, tc{lexeme: "}", token: tkRightBrace, depth: depth})
				depth--
			case '=':
				tokens = append(tokens, tc{lexeme: "=", token: tkEqual, depth: depth})
			case ',':
				tokens = append(tokens, tc{lexeme: ",", token: tkComma, depth: depth})
			case '"':
				tokens = append(tokens, tc{lexeme: "\"", token: tkQuote, depth: depth})
			case '_':
				tokens = append(tokens, tc{lexeme: "_", token: tkUnderscore, depth: depth})
			case ':':
				tokens = append(tokens, tc{lexeme: ":", token: tkColon, depth: depth})
			case '/':
				tokens = append(tokens, tc{lexeme: "/", token: tkForwardSlash, depth: depth})
			case '\\':
				tokens = append(tokens, tc{lexeme: "\\", token: tkBackslash, depth: depth})
			default:
				if unicode.IsLetter(char) || unicode.IsNumber(char) {
					tokens = append(tokens, tc{lexeme: string(char), token: tkLexeme, depth: depth})
				}
			}

			cursor++
		}

	}
	tokens = extractStrings(tokens)

	return tokens
}

func extractStrings(tokens []tc) []tc {
	tks := make([]tc, 0)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].token == tkLexeme {
			var newStr string
			var k int
			quoted := false

			if i > 0 && tokens[i-1].token == tkQuote {
				for j := i; j < len(tokens); j++ {
					if tokens[j].token != tkQuote {
						newStr += tokens[j].lexeme
					} else {
						k = j
						break
					}
				}
				quoted = true
			} else {
				for j := i; j < len(tokens); j++ {
					if tokens[j].token == tkLexeme || tokens[j].token == tkUnderscore {
						newStr += tokens[j].lexeme
					} else {
						k = j - 1 // Step back one to include the non-lexeme token
						break
					}
				}
			}

			tks = append(tks, tc{lexeme: newStr, token: tkLexeme, depth: tokens[i].depth})
			if quoted {
				tks = append(tks, tc{lexeme: "\"", token: tkQuote, depth: tokens[i].depth})
			}
			i = k
		} else {
			tks = append(tks, tokens[i])
		}
	}
	return tks
}
