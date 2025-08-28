package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/iamfat/docker-genee/internal/registry"
	"github.com/spf13/cobra"
)

var (
	platformFilter string
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "查看genee的镜像",
	Long:  `查看私有registry docker.genee.cn中的镜像列表`,
	RunE:  runImages,
}

func init() {
	rootCmd.AddCommand(imagesCmd)
	geneeCmd.AddCommand(imagesCmd)
	
	// 添加平台过滤参数
	imagesCmd.Flags().StringVar(&platformFilter, "platform", "", "过滤指定平台的镜像 (如: amd64, arm64)")
}

func runImages(cmd *cobra.Command, args []string) error {
	// 创建registry客户端
	client := registry.NewClient(registryURL)
	
	// 检查是否有有效的认证信息
	if !client.HasValidCredentials() {
		return fmt.Errorf("请先登录，使用 'docker genee login' 命令")
	}
	
	// 获取镜像列表
	images, err := client.ListImages(platformFilter)
	if err != nil {
		return fmt.Errorf("获取镜像列表失败: %v", err)
	}
	
	if len(images) == 0 {
		fmt.Println("没有找到镜像")
		return nil
	}
	
	// 使用tabwriter格式化输出
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "REPOSITORY\tTAG\tDIGEST\tPLATFORM\tSIZE\tCREATED")
	
	for _, img := range images {
		// 截断过长的仓库名
		repo := img.Repository
		if len(repo) > 30 {
			repo = repo[:27] + "..."
		}
		
		// 截断过长的标签名
		tag := img.Tag
		if len(tag) > 20 {
			tag = tag[:17] + "..."
		}
		
		// 格式化平台信息
		platforms := strings.Join(img.Platforms, ", ")
		if platforms == "" {
			platforms = "unknown"
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			repo,
			tag,
			img.Digest[:12], // 只显示前12位
			platforms,
			img.Size,
			img.Created)
	}
	
	w.Flush()
	
	fmt.Printf("\n总计: %d 个镜像\n", len(images))
	return nil
}
