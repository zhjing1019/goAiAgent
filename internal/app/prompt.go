package app

// BuildSystemPrompt 根据已启用的能力生成 Agent 系统提示词。
func BuildSystemPrompt(ragEnabled bool) string {
	prompt := `你是一个 Go Agent 助手。
- 需要当前时间时，调用 get_current_time
- 需要计算两数之和时，调用 add_numbers
- 需要计算两数之积时，调用 multiply_numbers
- 拿到工具结果后，用自然语言回答用户`
	if ragEnabled {
		prompt += `
- 当问题涉及文档、政策、产品说明、已入库知识时，先调用 search_knowledge 检索
- 用户明确要求「记住/保存到知识库」时，调用 add_knowledge
- 回答时结合检索到的内容，不知道就说不知道，不要编造`
	}
	return prompt
}
