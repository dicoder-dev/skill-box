package router

import (
	"ginp-api/pkg/ginp"
	"regexp"
	"strings"
)

// findRouterByPath 根据路径查找对应的路由项
func findRouterByPath(path string) *ginp.RouterItem {
	routers := ginp.GetAllRouter()
	for _, router := range routers {
		if matchPath(router.Path, path) {
			return &router
		}
	}
	return nil
}

// matchPath 匹配路径，支持Gin的路由规则
// 支持路径参数如 /user/:id 和通配符如 /user/*
func matchPath(pattern, path string) bool {
	// 如果模式包含路径参数或通配符，使用正则匹配
	if strings.Contains(pattern, ":") || strings.Contains(pattern, "*") {
		return matchPattern(pattern, path)
	}
	// 否则使用精确匹配
	return pattern == path
}

// matchPattern 使用正则表达式匹配路径模式
func matchPattern(pattern, path string) bool {
	// 将Gin路由模式转换为正则表达式
	regexPattern := convertToRegex(pattern)

	// 编译正则表达式
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		// 如果正则编译失败，回退到精确匹配
		return pattern == path
	}

	// 匹配路径
	return regex.MatchString(path)
}

// convertToRegex 将Gin路由模式转换为正则表达式
func convertToRegex(pattern string) string {
	// 先处理路径参数 :param
	paramRegex := regexp.MustCompile(`:([^/]+)`)
	pattern = paramRegex.ReplaceAllString(pattern, `[^/]+`)

	// 处理通配符 *
	pattern = strings.ReplaceAll(pattern, "*", ".*")

	// 转义其他正则特殊字符
	pattern = regexp.QuoteMeta(pattern)

	// 恢复路径参数和通配符的正则表达式
	pattern = strings.ReplaceAll(pattern, `\[^/\]\+`, `[^/]+`)
	pattern = strings.ReplaceAll(pattern, `\.\*`, `.*`)

	// 确保匹配整个路径
	return "^" + pattern + "$"
}
