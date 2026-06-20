package gen

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// ReplaceAndWriteTemplate 传入一个tmpl模板文件路径，一个占位符文本，一个真实值文本，输出文件路径四个参数。
// 实现读取tmp模板文件的值并将占位符本替换成传入的真实值，将最终替换好的文本写入到传入的输出文件路径中。
func ReplaceAndWriteTemplate(tmplFilePath string, outputFilePath string, placeholderMap map[string]string) error {
	//如果文件outputFilePath存在则跳过
	if _, err := os.Stat(outputFilePath); err == nil {
		println("文件", outputFilePath, "已存在，跳过生成  skin")
		return nil
	}
	//判断路径outputFilePath所在目录是否存在，不存在则先创建目录
	outputFileDir := filepath.Dir(outputFilePath)
	_, err := os.Stat(outputFileDir)
	if os.IsNotExist(err) {
		// 目录不存在，创建目录
		err := os.MkdirAll(outputFileDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	tmplContent, err := ioutil.ReadFile(tmplFilePath)
	if err != nil {
		return err
	}

	content := string(tmplContent)
	for placeholder, value := range placeholderMap {
		processedContent := strings.Replace(content, placeholder, value, -1)
		content = processedContent
	}

	err = ioutil.WriteFile(outputFilePath, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}

// 读取一个文件
func replaceTextInFile(filePath, oldText, newText string) error {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 文件不存在，创建文件
		_, err := os.Create(filePath)
		if err != nil {
			return err
		}
	}

	// 读取文件内容
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// 替换文本
	newContent := strings.ReplaceAll(string(content), oldText, newText)

	// 将替换后的内容写回文件
	err = ioutil.WriteFile(filePath, []byte(newContent), 0666)
	if err != nil {
		return err
	}

	return nil
}

// AppendContentToFile  向一个文件中追加替换内容
func AppendContentToFile(filePath, appendContent string) error {
	//如果指定文件不存在会自动创建
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(appendContent); err != nil {
		return err
	}
	return nil
}
