package main

import (
	"fmt"

	_ "ginp-api/internal/gapi/controller/skillbox/cdesktop"
	"ginp-api/pkg/ginp"
)

func main() {
	for _, r := range ginp.GetAllRouter() {
		fmt.Printf("%-6s %s\n", r.HttpType, r.Path)
	}
}
