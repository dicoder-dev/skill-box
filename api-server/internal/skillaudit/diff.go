package skillaudit

// diffOp 内部用,表示一次回溯的单元(避免到处写匿名结构体)。
type diffOp struct {
	kind string // context / added / removed
	ln   string
	lIdx int // 1-based
	rIdx int // 1-based
}

// LinesDiff 行级 diff:基于 LCS 的 Myers 简化版,产出 context / added / removed。
// 输入:两个字符串(按 \n 切行),输出 DiffLine 列表,context 行只出现在被改动行附近 1 行
// 范围内(避免长 unchanged 拖长输出)。
func LinesDiff(left, right string) []DiffLine {
	lLines := splitLines(left)
	rLines := splitLines(right)
	dp := lcsTable(lLines, rLines)
	return walkLCS(lLines, rLines, dp)
}

// lcsTable 构造 LCS 长度表(dp[i][j] = left[:i] vs right[:j] 的 LCS 长度)。
func lcsTable(left, right []string) [][]int {
	rows, cols := len(left), len(right)
	dp := make([][]int, rows+1)
	for i := range dp {
		dp[i] = make([]int, cols+1)
	}
	for i := 1; i <= rows; i++ {
		for j := 1; j <= cols; j++ {
			if left[i-1] == right[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	return dp
}

// walkLCS 从 dp 表回溯,产出 DiffLine(已折叠 context)。
func walkLCS(left, right []string, dp [][]int) []DiffLine {
	ops := make([]diffOp, 0, len(left)+len(right))
	i, j := len(left), len(right)
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && left[i-1] == right[j-1]:
			ops = append(ops, diffOp{kind: "context", ln: left[i-1], lIdx: i, rIdx: j})
			i--
			j--
		case j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]):
			ops = append(ops, diffOp{kind: "added", ln: right[j-1], rIdx: j})
			j--
		default:
			ops = append(ops, diffOp{kind: "removed", ln: left[i-1], lIdx: i})
			i--
		}
	}
	// 反向
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}
	collapsed := collapseContext(ops)
	out := make([]DiffLine, 0, len(collapsed))
	for _, o := range collapsed {
		dl := DiffLine{Kind: o.kind, Text: o.ln}
		if o.lIdx > 0 {
			dl.LeftNo = o.lIdx
		}
		if o.rIdx > 0 {
			dl.RightNo = o.rIdx
		}
		out = append(out, dl)
	}
	return out
}

func collapseContext(ops []diffOp) []diffOp {
	const ctxLines = 1 // 紧邻 changed 行的 context 行数
	changed := make([]bool, len(ops))
	for i, o := range ops {
		if o.kind != "context" {
			changed[i] = true
		}
	}
	keep := make([]bool, len(ops))
	for i := range ops {
		if changed[i] {
			keep[i] = true
			continue
		}
		for k := 1; k <= ctxLines && i-k >= 0; k++ {
			if changed[i-k] {
				keep[i] = true
				break
			}
		}
		if keep[i] {
			continue
		}
		for k := 1; k <= ctxLines && i+k < len(ops); k++ {
			if changed[i+k] {
				keep[i] = true
				break
			}
		}
	}
	out := make([]diffOp, 0, len(ops))
	for i, o := range ops {
		if keep[i] {
			out = append(out, o)
			continue
		}
		// 在两个 keep 之间塞一个 "..." 占位(只在 gap > 0 时)
		if i > 0 && keep[i-1] && i+1 < len(ops) && keep[i+1] {
			out = append(out, diffOp{kind: "context", ln: "..."})
		}
	}
	return out
}
