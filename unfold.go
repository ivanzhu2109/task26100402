package vcard

// unfoldLines 将输入字节流按 CRLF/LF 切分为逻辑行，
// 并展开续行（行首空格或制表符）。展开在字节层面进行，
// 允许折行落在 UTF-8 多字节字符内部；仅移除换行及紧随其后的
// 一个空白字符，连续续行依次展开。
func unfoldLines(data []byte) []string {
	var physical [][]byte
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] != '\n' {
			continue
		}
		line := data[start:i]
		// 去掉行末 CR（兼容 CRLF 与裸 LF）
		if n := len(line); n > 0 && line[n-1] == '\r' {
			line = line[:n-1]
		}
		physical = append(physical, line)
		start = i + 1
	}
	// 最后一行没有换行符也接受
	if start < len(data) {
		physical = append(physical, data[start:])
	}

	var logical []string
	for _, line := range physical {
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && len(logical) > 0 {
			// 续行：去掉一个空白字符后拼接
			logical[len(logical)-1] += string(line[1:])
			continue
		}
		logical = append(logical, string(line))
	}
	return logical
}
