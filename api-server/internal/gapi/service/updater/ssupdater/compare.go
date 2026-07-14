package ssupdater

import (
	"fmt"

	"golang.org/x/mod/semver"
)

// 比较结论。
const (
	StatusUpToDate    = "upToDate"
	StatusAvailable   = "available"
	StatusMustUpdate  = "mustUpdate"
	StatusIncomparable = "incomparable" // 一侧不是有效 semver
)

// Compare 比对 local / remote,返回状态字符串。
// local / remote 都必须以 v 前缀或不带前缀的纯版本号传入(MVP 阶段不强制 v 前缀,
// 缺 v 视为去掉前导 v 后按 semver 处理)。
//
// 规则:
//   - 解析不出 semver → StatusIncomparable(前端不展示升级,降级为"已是最新")
//   - remote > local → StatusAvailable
//   - remote == local → StatusUpToDate
//   - remote < local → StatusUpToDate(降级场景,前端不弹升级)
//   - remote < minSupported 且 local < remote → StatusMustUpdate(强制升级)
func Compare(local, remote, minSupported string) string {
	l := normalize(local)
	r := normalize(remote)
	if !semver.IsValid(l) || !semver.IsValid(r) {
		return StatusIncomparable
	}
	cmp := semver.Compare(l, r)
	if cmp < 0 {
		// remote > local,看是否强制升级
		if semver.IsValid(normalize(minSupported)) && semver.Compare(l, normalize(minSupported)) < 0 {
			return StatusMustUpdate
		}
		return StatusAvailable
	}
	return StatusUpToDate
}

// normalize 把 MAJOR.MINOR.PATCH / vMAJOR.MINOR.PATCH 都映射到 semver 兼容的 v 前缀。
// 空串直接返回,IsValid 会拒掉。
func normalize(v string) string {
	v = trim(v)
	if v == "" {
		return ""
	}
	if v[0] != 'v' {
		return "v" + v
	}
	return v
}

func trim(s string) string {
	// 去前后空白,简单 trim
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}
	if i >= j {
		return ""
	}
	// 拒掉类似 "v1.2.3a" 这种非数字后缀(可能误判),但保留 -prerelease (semver 自带支持)
	for _, c := range s[i:j] {
		if c == '/' || c == '\\' || c == '?' || c == '#' {
			return ""
		}
	}
	return s[i:j]
}

// CompareError 把 incomparable 当 error 抛出去,controller 调用层需要明确知道。
func CompareError(local, remote string) error {
	if Compare(local, remote, "") == StatusIncomparable {
		return fmt.Errorf("updater: incomparable versions local=%q remote=%q", local, remote)
	}
	return nil
}
