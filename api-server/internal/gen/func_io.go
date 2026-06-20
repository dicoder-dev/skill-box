package gen

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type handler func(inputContent string)

func Input(desc string, afterHandler handler) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(desc)
	input, err := reader.ReadString('\n') // 读取用户输入直到换行符
	if err != nil {
		panic("input error:" + err.Error())
	}
	if afterHandler != nil {
		afterHandler(input)
	}
	return strings.TrimSpace(input)
}
