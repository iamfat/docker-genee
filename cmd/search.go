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
	platform string
	limit    int
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "搜索基理镜像",
	Long: `搜索基理科技镜像源中的镜像，支持通配符和平台限制。

示例:
  docker genee search ph*          # 搜索以ph开头的镜像
  docker genee search php:a*       # 搜索php镜像中以a开头的标签
  docker genee search ph* --platform arm64  # 限制平台为arm64`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	geneeCmd.AddCommand(searchCmd)
	
	// 搜索相关标志
	searchCmd.Flags().StringVar(&platform, "platform", "", "限制搜索的平台 (如: linux/amd64, linux/arm64, amd64, arm64)")
	searchCmd.Flags().IntVar(&limit, "limit", 100, "搜索结果数量限制")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	
	// 创建registry客户端
	client := registry.NewClient(registryURL)
	
	// 检查是否有有效的认证信息
	if !client.HasValidCredentials() {
		return fmt.Errorf("请先登录，使用 'docker genee login' 命令")
	}
	
	// 搜索镜像
	results, err := client.SearchImages(query, platform, limit)
	if err != nil {
		return fmt.Errorf("搜索镜像失败: %v", err)
	}
	
	if len(results) == 0 {
		fmt.Printf("没有找到匹配 '%s' 的镜像", query)
		if platform != "" {
			fmt.Printf(" (平台: %s)", platform)
		}
		fmt.Println()
		return nil
	}
	
	// 使用tabwriter格式化输出
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "REPOSITORY\tTAG\tDIGEST\tPLATFORM\tSIZE\tCREATED")
	
	for _, result := range results {
		// 获取真实的平台信息（而不是硬编码的平台列表）
		platforms := result.Platforms
		if len(platforms) == 0 {
			platforms = []string{"unknown"}
		}
		
		// 截断过长的仓库名
		repo := result.Name
		if len(repo) > 30 {
			repo = repo[:27] + "..."
		}
		
		// 截断过长的摘要
		digest := result.Digest
		if len(digest) > 12 {
			digest = digest[:12]
		}
		
		// 如果有匹配的标签，为每个标签创建单独的行
		if len(result.MatchedTags) > 0 {
			for _, tag := range result.MatchedTags {
				// 截断过长的标签名
				tagDisplay := tag
				if len(tagDisplay) > 20 {
					tagDisplay = tagDisplay[:17] + "..."
				}
				
				// 为每个标签获取真实的平台信息
				tagPlatforms := getTagPlatforms(client, result.Name, tag)
				platformDisplay := strings.Join(tagPlatforms, ", ")
				if platformDisplay == "" {
					platformDisplay = "unknown"
				}
				
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					repo,
					tagDisplay,
					digest,
					platformDisplay,
					result.Size,
					result.Created)
			}
		} else {
			// 没有匹配的标签，显示最新标签
			tagDisplay := result.LatestTag
			if len(tagDisplay) > 20 {
				tagDisplay = tagDisplay[:17] + "..."
			}
			
			// 为最新标签获取真实的平台信息
			tagPlatforms := getTagPlatforms(client, result.Name, result.LatestTag)
			platformDisplay := strings.Join(tagPlatforms, ", ")
			if platformDisplay == "" {
				platformDisplay = "unknown"
			}
			
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				repo,
				tagDisplay,
				digest,
				platformDisplay,
				result.Size,
				result.Created)
		}
	}
	
	w.Flush()
	
	// 计算总行数（包括每个匹配的标签）
	totalLines := 0
	for _, result := range results {
		if len(result.MatchedTags) > 0 {
			totalLines += len(result.MatchedTags)
		} else {
			totalLines++
		}
	}
	
	fmt.Printf("\n找到 %d 个匹配的镜像，共 %d 行", len(results), totalLines)
	if platform != "" {
		fmt.Printf(" (平台: %s)", platform)
	}
	fmt.Println()
	
	return nil
}

// getTagPlatforms 获取指定标签的真实平台信息
func getTagPlatforms(client *registry.Client, repository, tag string) []string {
	// 使用与 images.go 相同的方法获取平台信息
	platforms := client.GetImagePlatforms(repository, tag)
	if len(platforms) == 0 {
		return []string{"unknown"}
	}
	return platforms
}
