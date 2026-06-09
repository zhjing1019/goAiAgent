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
	// 如果文本长度小于等于最大字符数，则直接返回文本
	if utf8.RuneCountInString(text) <= maxRunes {
		return []string{text}
	}
	// 如果文本长度大于最大字符数，则将文本切成多个片段
	var out []string
	// 创建一个空数组，用于存储片段
	var buf []rune
	// 遍历文本，将文本切成多个片段
	for _, r := range text {
		// 将字符追加到缓冲区
		buf = append(buf, r)
		// 如果缓冲区长度大于等于最大字符数，则将缓冲区中的文本追加到结果数组中
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
	// 如果字符串长度大于0，并且字符串的第一个字符是空格、换行符、回车符或制表符，则将字符串的第一个字符去掉
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\r' || s[0] == '\t') {
		s = s[1:]
	}
	// 如果字符串长度大于0，并且字符串的最后一个字符是空格、换行符、回车符或制表符，则将字符串的最后一个字符去掉
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != ' ' && last != '\n' && last != '\r' && last != '\t' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}
