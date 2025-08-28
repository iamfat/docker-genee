package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

// Metadata 结构体定义插件的元数据
type Metadata struct {
	// SchemaVersion 描述这个结构的版本。必需，必须是 "0.1.0"
	SchemaVersion string `json:",omitempty"`
	// Vendor 是插件供应商的名称。必需
	Vendor string `json:",omitempty"`
	// Version 是这个插件的可选版本
	Version string `json:",omitempty"`
	// ShortDescription 应该适合单行帮助消息
	ShortDescription string `json:",omitempty"`
	// URL 是指向插件主页的指针
	URL string `json:",omitempty"`
}

var metadataCmd = &cobra.Command{
	Use:   "docker-cli-plugin-metadata",
	Short: "返回插件元数据",
	Long:  `返回Docker CLI插件所需的元数据信息`,
	RunE:  runMetadata,
	Hidden: true, // 隐藏这个命令，不在帮助中显示
}

func init() {
	rootCmd.AddCommand(metadataCmd)
}

func runMetadata(cmd *cobra.Command, args []string) error {
	metadata := Metadata{
		SchemaVersion:    "0.1.0",
		Vendor:           "基理科技",
		Version:          Version,
		ShortDescription: "基理科技镜像源操作插件",
		URL:              "https://github.com/iamfat/docker-genee",
	}

	// 将元数据输出为JSON格式
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", " ")
	return encoder.Encode(metadata)
}
