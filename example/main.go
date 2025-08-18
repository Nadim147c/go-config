package main

import (
	"fmt"

	"github.com/Nadim147c/go-config"
)

func main() {
	config.AddFile("./test/config.json")
	config.ReadConfig()
	portStr := config.GetStringMust("app.port")
	fmt.Println(":" + portStr)
}
