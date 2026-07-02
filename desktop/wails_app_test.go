// 2026-07-02 增:ParseAspectRatio 单元测试。覆盖"合法 + 非法"两类输入,
// 并验证函数与 NewApp / resizePrimaryToScreenRatio 的实际行为契约:
//
//   - 空串 → (0, 0)(调用方走原独立 widthRatio / heightRatio 路径)
//   - 合法 W:H 字符串 → 解析出正整数对(后续按 aspect 反推窗口高)
//   - 非法输入(单边 0、负数、非数字、缺冒号、3+ 段)→ (0, 0)(安全降级)
//
// 该函数是 desktop 包内 helper,被 NewApp 与 Run 链路引用,放在 desktop_test 包
// (white-box)以便白盒单测。
package desktop

import "testing"

func TestParseAspectRatio(t *testing.T) {
	cases := []struct {
		in  string
		w   int
		h   int
		why string
	}{
		// 合法输入
		{"16:9", 16, 9, "经典宽屏 16:9"},
		{"4:3", 4, 3, "传统 4:3"},
		{"21:9", 21, 9, "超宽 21:9"},
		{"1:1", 1, 1, "正方形"},
		{" 16 : 9 ", 16, 9, "带空格"},
		{"256:135", 256, 135, "大数字"},

		// 非法输入 → (0, 0) 降级
		{"", 0, 0, "空串"},
		{"   ", 0, 0, "仅空白"},
		{":", 0, 0, "两个冒号空数字"},
		{"16:", 0, 0, "右边空"},
		{":9", 0, 0, "左边空"},
		{"0:9", 0, 0, "左边 0"},
		{"16:0", 0, 0, "右边 0"},
		{"-16:9", 0, 0, "负数"},
		{"16:-9", 0, 0, "右边负数"},
		{"abc:9", 0, 0, "非数字"},
		{"16:xyz", 0, 0, "右边非数字"},
		{"16", 0, 0, "缺冒号"},
		{"16:9:1", 0, 0, "3 段"},
		{"16/9", 0, 0, "错误分隔符"},
	}
	for _, c := range cases {
		gotW, gotH := ParseAspectRatio(c.in)
		if gotW != c.w || gotH != c.h {
			t.Errorf("ParseAspectRatio(%q) = (%d, %d), want (%d, %d) — %s",
				c.in, gotW, gotH, c.w, c.h, c.why)
		}
	}
}
