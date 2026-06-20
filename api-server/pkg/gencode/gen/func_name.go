package gen

import (
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// NameToLine 下划线命名
func NameToLine(camel string) string {
	if  strings.Contains(camel, "-") {
		return strings.ReplaceAll(camel, "-", "_")
	}
	var snake []rune
	for i, c := range camel {
		if unicode.IsUpper(c) {
			if i > 0 && (i+1 < len(camel) && unicode.IsLower(rune(camel[i+1])) || unicode.IsLower(rune(camel[i-1]))) {
				snake = append(snake, '_')
			}
			snake = append(snake, unicode.ToLower(c))
		} else {
			snake = append(snake, c)
		}
	}
	return string(snake)
}

// NameToCameSmall 小驼峰命名
func NameToCameSmall(s string) string {
	if strings.Contains(s, "_") || strings.Contains(s, "-") { //如果原来是下划线命名法
		splitor := "_"
		if strings.Contains(s, "-") {
			splitor = "-"
		}
		words := strings.Split(s, splitor)
		for i := 0; i < len(words); i++ {
			words[i] = strings.ToLower(words[i])
			if i > 0 {
				words[i] = cases.Title(language.English).String(words[i])
			}
		}
		return strings.Join(words, "")
	}

	var words []string
	for i := 0; i < len(s); i++ {
		if i > 0 && (unicode.IsUpper(rune(s[i])) || !(unicode.IsLetter(rune(s[i])) || unicode.IsDigit(rune(s[i])))) {
			// 如果当前字符是大写字母或者非字母数字字符，则将前面的部分作为一个单词
			words = append(words, s[:i])
			s = s[i:]
			i = 0
		}
	}
	words = append(words, s)
	for i := 0; i < len(words); i++ {
		if i == 0 {
			words[i] = strings.ToLower(words[i])
		} else {
			words[i] = cases.Title(language.English).String(words[i])
		}
	}

	return strings.Join(words, "")
}

// NameToCameBig 大驼峰命名法
func NameToCameBig(line string) string {
	var camel []rune
	splitor := "_"
	if strings.Contains(line, "-") {
		splitor = "-"
	}
	words := strings.Split(line, splitor)

	for _, word := range words {
		firstChar := word[0]
		restChars := word[1:]

		camel = append(camel, unicode.ToUpper(rune(firstChar)))
		camel = append(camel, []rune(restChars)...)
	}

	return string(camel)
}

// NameToKebab 中线命名法
func NameToKebab(line string) string {
	lineName := NameToLine(line)
	return strings.ReplaceAll(lineName, "_", "-")
}

// 全小写命名
func NameToAllSmall(line string) string {
	lineName := NameToLine(line)
	return strings.ReplaceAll(lineName, "_", "")
}
