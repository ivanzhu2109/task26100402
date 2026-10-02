package vcard

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// 场景二：备注中出现边界字样、修改单个字段后的往返，
// 验证名片边界识别、未知属性保留及 75 字节折行限制。
func TestBoundaryModifyAndRoundTrip(t *testing.T) {
	// NOTE 解码后包含 "BEGIN:VCARD" 与换行；另有一行未知属性
	// 折行后的续行内容以边界字样开头（续行不算边界）。
	noteText := "前面\nBEGIN:VCARD\nEND:VCARD\n后面"
	longUnknown := "X-LONG-NOTE:这是" +
		"一段很长很长的用于测试折行的未知属性内容需要超过七十五字节才会触发折行逻辑" +
		"所以继续填充更多中文字符让它足够长到必须折行才行"
	input := "BEGIN:VCARD\r\n" +
		"VERSION:3.0\r\n" +
		"FN:测试用户\r\n" +
		"TEL;TYPE=CELL:111\r\n" +
		"TEL;TYPE=WORK:222\r\n" +
		"NOTE:" + escapeText(noteText) + "\r\n" +
		longUnknown + "\r\n" +
		"END:VCARD\r\n" +
		"BEGIN:VCARD\r\n" +
		"FN:第二张名片\r\n" +
		"END:VCARD\r\n"

	cards, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	// 备注中的边界字样不能成为边界：应解析出恰好 2 张名片
	if len(cards) != 2 {
		t.Fatalf("名片数量 = %d, 期望 2（备注中的边界字样被误识别）", len(cards))
	}
	c := cards[0]
	if got := c.Get("NOTE").Text(); got != noteText {
		t.Errorf("NOTE = %q, 期望 %q", got, noteText)
	}

	// 记录修改前各属性的展开内容快照
	before := make([]string, len(c.Props))
	for i, p := range c.Props {
		before[i] = writeProperty(p)
	}

	// 只修改第一个 TEL
	c.All("TEL")[0].SetText("999")

	// 未知属性展开后的内容逐字节不变
	if got := c.Get("X-LONG-NOTE").Value; got != longUnknown[len("X-LONG-NOTE:"):] {
		t.Errorf("未知属性内容被改变")
	}

	out := Marshal(cards)

	// 75 字节折行限制：每条物理行（不含 CRLF）不超过 75 字节，
	// 且每条物理行自身是合法 UTF-8 前缀（未从字符内部切断）。
	physical := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	for _, line := range physical {
		if line == "" {
			continue
		}
		content := line
		if line[0] == ' ' {
			content = line[1:]
		}
		if len(line) > maxLineBytes {
			t.Errorf("物理行超过 75 字节 (%d): %q", len(line), line)
		}
		if !utf8.ValidString(content) {
			t.Errorf("物理行从 UTF-8 字符内部切断: %q", line)
		}
	}
	// 输出统一使用 CRLF
	if strings.Contains(string(out), "\n") &&
		len(strings.ReplaceAll(string(out), "\r\n", "")) !=
			len(strings.ReplaceAll(strings.ReplaceAll(string(out), "\r\n", ""), "\n", "")) {
		t.Errorf("存在裸 LF 换行")
	}

	// 往返：再次解析，除被修改的 TEL 外，其余属性逐字节一致
	cards2, err := Parse(out)
	if err != nil {
		t.Fatalf("往返 Parse 失败: %v", err)
	}
	if len(cards2) != 2 {
		t.Fatalf("往返名片数量 = %d, 期望 2", len(cards2))
	}
	c2 := cards2[0]
	if len(c2.Props) != len(c.Props) {
		t.Fatalf("往返属性数 = %d, 期望 %d", len(c2.Props), len(c.Props))
	}
	for i, p := range c2.Props {
		want := before[i]
		if p.Name == "TEL" && p.Params[0].Values[0] == "CELL" {
			want = "TEL;TYPE=CELL:999"
		}
		if got := writeProperty(p); got != want {
			t.Errorf("属性[%d] = %q, 期望 %q", i, got, want)
		}
	}
	if got := c2.Get("NOTE").Text(); got != noteText {
		t.Errorf("往返 NOTE = %q, 期望 %q", got, noteText)
	}
	// 第二张名片不受影响
	if got := cards2[1].Get("FN").Text(); got != "第二张名片" {
		t.Errorf("第二张名片 FN = %q", got)
	}
}
