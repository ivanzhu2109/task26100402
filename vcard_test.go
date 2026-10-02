package vcard

import (
	"testing"
)

// 场景一：结构分隔符与转义标点混合、带引号参数和中文折行。
func TestParseEscapedQuotedAndChineseFolding(t *testing.T) {
	// 手工构造输入：中文被折行从 UTF-8 字符内部切断，
	// 参数含引号包裹的分号/冒号/逗号，值含转义标点。
	name := "一个很长很长的中文名字需要被折行处理才行"
	// 找到名字中间某个多字节字符的中间字节位置，在那里折行
	raw := "FN;CHARSET=UTF-8:" + name
	cut := 30 // 落在某个中文字符的多字节序列内部
	if cut >= len(raw) || raw[cut]&0xC0 == 0 {
		t.Fatalf("cut 位置未落在多字节字符内部: %d", cut)
	}
	input := "BEGIN:VCARD\r\n" +
		"VERSION:3.0\r\n" +
		raw[:cut] + "\r\n " + raw[cut:] + "\r\n" +
		`N:张\;三;小\,明;;博士;` + "\r\n" +
		`ORG:ACME\, 公司;研发\;部;;` + "\r\n" +
		`TEL;TYPE="work,voice";X-REF="a:b";C=d,e:+86-138-0000-0000` + "\r\n" +
		`EMAIL;TYPE=HOME:zhang@example.com` + "\r\n" +
		`ADR;TYPE=HOME:;;北京市\;海淀区;中关村\,大街1号;;100080;中国` + "\r\n" +
		`NOTE:第一行\n第二行\, 含逗号\; 含分号\\含反斜杠` + "\r\n" +
		`CATEGORIES:朋友\,同事,,VIP` + "\r\n" +
		`item1.X-UNKNOWN;P="a;b":值:含冒号;含分号` + "\n" + // 裸 LF 混用
		"END:VCARD" // 末尾无换行符

	cards, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("名片数量 = %d, 期望 1", len(cards))
	}
	c := cards[0]

	// FN：跨 UTF-8 字符的折行展开后应完整还原
	if got := c.Get("fn").Text(); got != name {
		t.Errorf("FN = %q, 期望 %q", got, name)
	}
	// N：5 个分量，转义的分号/逗号属于内容，空分量保留
	n := c.Get("N").Structured(NComponents)
	wantN := [][]string{{"张;三"}, {"小,明"}, {""}, {"博士"}, {""}}
	assertMatrix(t, "N", n, wantN)
	// ORG：按分号划分，尾部空分量保留
	org := c.Get("ORG").Components()
	assertSlice(t, "ORG", org, []string{"ACME, 公司", "研发;部", "", ""})
	// TEL：带引号参数中的逗号/冒号不参与分隔，重复参数保留
	tel := c.Get("TEL")
	if tel.Text() != "+86-138-0000-0000" {
		t.Errorf("TEL = %q", tel.Text())
	}
	wantParams := []Param{
		{Name: "TYPE", Values: []string{"work,voice"}},
		{Name: "X-REF", Values: []string{"a:b"}},
		{Name: "C", Values: []string{"d", "e"}},
	}
	if len(tel.Params) != len(wantParams) {
		t.Fatalf("TEL 参数数 = %d, 期望 %d", len(tel.Params), len(wantParams))
	}
	for i, wp := range wantParams {
		if tel.Params[i].Name != wp.Name {
			t.Errorf("参数[%d].Name = %q, 期望 %q", i, tel.Params[i].Name, wp.Name)
		}
		assertSlice(t, "参数值", tel.Params[i].Values, wp.Values)
	}
	// ADR：7 个分量，转义标点还原
	adr := c.Get("ADR").Structured(ADRComponents)
	wantADR := [][]string{{""}, {""}, {"北京市;海淀区"}, {"中关村,大街1号"}, {""}, {"100080"}, {"中国"}}
	assertMatrix(t, "ADR", adr, wantADR)
	// NOTE：\n 还原为换行，转义标点还原
	wantNote := "第一行\n第二行, 含逗号; 含分号\\含反斜杠"
	if got := c.Get("NOTE").Text(); got != wantNote {
		t.Errorf("NOTE = %q, 期望 %q", got, wantNote)
	}
	// CATEGORIES：空列表项保留
	assertSlice(t, "CATEGORIES", c.Get("CATEGORIES").Items(),
		[]string{"朋友,同事", "", "VIP"})
	// 未知属性：组名前缀与原始值保留
	unk := c.Get("X-UNKNOWN")
	if unk == nil || unk.Group != "item1" {
		t.Fatalf("未知属性组名丢失: %+v", unk)
	}
	if unk.Value != "值:含冒号;含分号" {
		t.Errorf("未知属性值 = %q", unk.Value)
	}

	// 写回后再次读取，语义应完全一致
	cards2, err := Parse(Marshal(cards))
	if err != nil {
		t.Fatalf("往返 Parse 失败: %v", err)
	}
	c2 := cards2[0]
	if got := c2.Get("FN").Text(); got != name {
		t.Errorf("往返 FN = %q", got)
	}
	assertMatrix(t, "往返 N", c2.Get("N").Structured(NComponents), wantN)
	assertSlice(t, "往返 ORG", c2.Get("ORG").Components(),
		[]string{"ACME, 公司", "研发;部", "", ""})
	assertMatrix(t, "往返 ADR", c2.Get("ADR").Structured(ADRComponents), wantADR)
	if got := c2.Get("NOTE").Text(); got != wantNote {
		t.Errorf("往返 NOTE = %q", got)
	}
	assertSlice(t, "往返 CATEGORIES", c2.Get("CATEGORIES").Items(),
		[]string{"朋友,同事", "", "VIP"})
	if got := c2.Get("X-UNKNOWN"); got == nil || got.Value != unk.Value || got.Group != "item1" {
		t.Errorf("往返未知属性不一致: %+v", got)
	}
}

func assertSlice(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s 长度 = %d (%v), 期望 %d (%v)", label, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, 期望 %q", label, i, got[i], want[i])
		}
	}
}

func assertMatrix(t *testing.T, label string, got, want [][]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s 分量数 = %d (%v), 期望 %d", label, len(got), got, len(want))
	}
	for i := range want {
		assertSlice(t, label+" 分量", got[i], want[i])
	}
}
