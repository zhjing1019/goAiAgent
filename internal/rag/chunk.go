package rag

import "unicode/utf8"

// SplitText 把长文本切成多个片段（简单按字符数切，RAG 入门够用）。
//
// maxRunes <= 0 时默认 500 字。
func SplitText(text string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = 500
	}
	text = trimSpace(text)
	if text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) <= maxRunes {
		return []string{text}
	}

	var out []string
	var buf []rune
	for _, r := range text {
		buf = append(buf, r)
		if len(buf) >= maxRunes {
			out = append(out, string(buf))
			buf = buf[:0]
		}
	}
	if len(buf) > 0 {
		out = append(out, string(buf))
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\r' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != ' ' && last != '\n' && last != '\r' && last != '\t' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}
