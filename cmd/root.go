package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	registryURL = "docker.genee.cn"
	configDir   string
	Version     = "1.0.3"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Short: "基理科技镜像源操作插件",
	Long: `
支持的功能：
- 登录到基理科技镜像源
- 查看镜像列表
- 搜索镜像（支持通配符和平台限制）`,
	SilenceErrors: true,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 如果没有子命令，显示帮助信息
		return cmd.Help()
	},
}

// geneeCmd 是主要的命令，包含所有子命令，用于支持 docker genee 的调用方式
var geneeCmd = &cobra.Command{
	Use:   "genee",
	Short: "基理科技镜像源操作",
	Long:  `基理科技镜像源的各种操作命令`,
	Hidden: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 设置配置目录
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取用户主目录失败: %v\n", err)
		os.Exit(1)
	}

	configDir = home + "/.docker-genee"

	// 将 genee 命令添加到根命令，支持 docker genee 的调用方式
	rootCmd.AddCommand(geneeCmd)

	// 全局标志 - 同时添加到 rootCmd 和 geneeCmd
	rootCmd.PersistentFlags().StringVar(&registryURL, "registry", registryURL, "镜像源地址 (默认: docker.genee.cn)")
	geneeCmd.PersistentFlags().StringVar(&registryURL, "registry", registryURL, "镜像源地址 (默认: docker.genee.cn)")
}
