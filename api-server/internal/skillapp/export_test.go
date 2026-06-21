package skillapp

// SemverCmpForTest 把包内私有 semverCmp 暴露给 *_test.go 用(白盒 export)。
func SemverCmpForTest(a, b string) int {
	return semverCmp(a, b)
}
