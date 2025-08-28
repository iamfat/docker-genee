package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/iamfat/docker-genee/internal/registry"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "登录到基理镜像源",
	Long:  `登录到基理科技私有镜像源`,
	RunE:  runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
	geneeCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	fmt.Printf("登录到 %s\n", registryURL)
	
	// 获取用户名
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("用户名: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("读取用户名失败: %v", err)
	}
	username = strings.TrimSpace(username)
	
	// 获取密码（隐藏输入）
	fmt.Print("密码: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("读取密码失败: %v", err)
	}
	fmt.Println() // 换行
	
	password := string(bytePassword)
	
	// 创建registry客户端
	client := registry.NewClient(registryURL)
	
	// 尝试登录
	if err := client.Login(username, password); err != nil {
		return fmt.Errorf("登录失败: %v", err)
	}
	
	// 保存认证信息
	if err := client.SaveCredentials(username, password); err != nil {
		return fmt.Errorf("保存认证信息失败: %v", err)
	}
	
	fmt.Println("登录成功！")
	return nil
}
