package main

import (
	"fmt"
	"os"

	"github.com/iamfat/docker-genee/cmd"
)

func main() {
	// 检查是否是版本查询
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("docker-genee version %s\n", cmd.Version)
			os.Exit(0)
		case "--help", "-h", "help":
			// 让 Cobra 处理帮助信息
			break
		}
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
