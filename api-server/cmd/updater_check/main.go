package main

import (
	"fmt"

	"ginp-api/pkg/ginp"
	_ "ginp-api/internal/gapi/controller/skillbox/cdesktop"
)

func main() {
	for _, r := range ginp.GetAllRouter() {
		fmt.Printf("%-6s %s\n", r.HttpType, r.Path)
	}
}
