package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Client 表示registry客户端
type Client struct {
	registryURL string
	httpClient  *http.Client
	credentials *Credentials
}

// Credentials 表示认证信息
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token,omitempty"`
	Expires  int64  `json:"expires,omitempty"`
}

// Image 表示镜像信息
type Image struct {
	Repository string   `json:"repository"`
	Tag        string   `json:"tag"`
	Digest     string   `json:"digest"`
	Size       string   `json:"size"`
	Created    string   `json:"created"`
	Platforms  []string `json:"platforms"`
}

// SearchResult 表示搜索结果
type SearchResult struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        int      `json:"tags"`
	Size        string   `json:"size"`
	Platforms   []string `json:"platforms"`
	Digest      string   `json:"digest"`
	Created     string   `json:"created"`
	LatestTag   string   `json:"latest_tag"`
	MatchedTags []string `json:"matched_tags"`
}

// Manifest 表示镜像清单
type Manifest struct {
	Digest  string `json:"digest"`
	Size    int64  `json:"size"`
	Created string `json:"created"`
}

// NewClient 创建新的registry客户端
func NewClient(registryURL string) *Client {
	return &Client{
		registryURL: registryURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login 登录到registry
func (c *Client) Login(username, password string) error {
	// 构建认证URL
	authURL := fmt.Sprintf("https://%s/v2/", c.registryURL)
	
	// 创建基本认证头
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	
	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "Basic "+auth)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("认证失败，状态码: %d", resp.StatusCode)
	}
	
	// 保存认证信息到Docker凭证存储
	if err := c.saveDockerCredentials(username, password); err != nil {
		return fmt.Errorf("保存到Docker凭证存储失败: %v", err)
	}
	
	// 同时保存到本地（向后兼容）
	c.credentials = &Credentials{
		Username: username,
		Password: password,
	}
	
	return nil
}

// SaveCredentials 保存认证信息到本地（向后兼容）
func (c *Client) SaveCredentials(username, password string) error {
	// 确保配置目录存在
	configDir := os.Getenv("HOME") + "/.docker-genee"
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}
	
	credentials := &Credentials{
		Username: username,
		Password: password,
	}
	
	data, err := json.Marshal(credentials)
	if err != nil {
		return err
	}
	
	configFile := filepath.Join(configDir, "credentials.json")
	return os.WriteFile(configFile, data, 0600)
}

// saveDockerCredentials 保存认证信息到Docker凭证存储
func (c *Client) saveDockerCredentials(username, password string) error {
	// 尝试多种方式保存到Docker凭证存储
	
	// 方法1: 使用docker-credential-helper
	if err := c.saveCredentialsToHelper(username, password); err == nil {
		return nil
	}
	
	// 方法2: 保存到Docker配置文件
	if err := c.saveCredentialsToDockerConfig(username, password); err == nil {
		return nil
	}
	
	return fmt.Errorf("无法保存到Docker凭证存储")
}

// saveCredentialsToHelper 使用凭证助手保存凭证
func (c *Client) saveCredentialsToHelper(username, password string) error {
	// 尝试不同的凭证助手
	helpers := []string{"docker-credential-helper", "docker-credential-desktop", "docker-credential-ecr-login"}
	
	for _, helper := range helpers {
		if err := c.trySaveCredentialsHelper(helper, username, password); err == nil {
			return nil
		}
	}
	
	return fmt.Errorf("无法使用凭证助手保存凭证")
}

// trySaveCredentialsHelper 尝试使用特定的凭证助手保存凭证
func (c *Client) trySaveCredentialsHelper(helper, username, password string) error {
	// 构建凭证数据
	creds := map[string]string{
		"Username": username,
		"Secret":   password,
	}
	
	// 序列化为JSON
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	
	// 使用凭证助手保存凭证
	cmd := exec.Command(helper, "store")
	cmd.Stdin = strings.NewReader(fmt.Sprintf("%s\n%s", c.registryURL, string(data)))
	
	return cmd.Run()
}

// saveCredentialsToDockerConfig 保存凭证到Docker配置文件
func (c *Client) saveCredentialsToDockerConfig(username, password string) error {
	configPath := os.Getenv("HOME") + "/.docker/config.json"
	
	// 读取现有配置
	var config struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &config)
	}
	
	// 确保Auths字段存在
	if config.Auths == nil {
		config.Auths = make(map[string]struct {
			Auth string `json:"auth"`
		})
	}
	
	// 编码认证信息
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	config.Auths[c.registryURL] = struct {
		Auth string `json:"auth"`
	}{
		Auth: auth,
	}
	
	// 保存配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0600)
}

// LoadCredentials 从Docker凭证存储加载认证信息
func (c *Client) LoadCredentials() error {
	// 尝试从Docker凭证存储获取认证信息
	creds, err := c.getDockerCredentials()
	if err != nil {
		// 如果无法从Docker获取，尝试从本地文件加载（向后兼容）
		return c.loadLocalCredentials()
	}
	
	c.credentials = creds
	return nil
}

// getDockerCredentials 从Docker凭证存储获取认证信息
func (c *Client) getDockerCredentials() (*Credentials, error) {
	// 尝试多种方式获取Docker凭证
	
	// 方法1: 使用docker-credential-helper
	if creds, err := c.getCredentialsFromHelper(); err == nil {
		return creds, nil
	}
	
	// 方法2: 从Docker配置文件读取
	if creds, err := c.getCredentialsFromDockerConfig(); err == nil {
		return creds, nil
	}
	
	return nil, fmt.Errorf("无法从Docker获取凭证")
}

// getCredentialsFromHelper 使用docker-credential-helper获取凭证
func (c *Client) getCredentialsFromHelper() (*Credentials, error) {
	// 尝试不同的凭证助手
	helpers := []string{"docker-credential-helper", "docker-credential-desktop", "docker-credential-ecr-login"}
	
	for _, helper := range helpers {
		if creds, err := c.tryCredentialHelper(helper); err == nil {
			return creds, nil
		}
	}
	
	return nil, fmt.Errorf("无法使用凭证助手获取凭证")
}

// tryCredentialHelper 尝试使用特定的凭证助手
func (c *Client) tryCredentialHelper(helper string) (*Credentials, error) {
	cmd := exec.Command(helper, "get")
	cmd.Stdin = strings.NewReader(c.registryURL)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	// 解析输出格式：{"Username":"user","Secret":"pass"}
	var creds struct {
		Username string `json:"Username"`
		Secret   string `json:"Secret"`
	}
	
	if err := json.Unmarshal(output, &creds); err != nil {
		return nil, err
	}
	
	if creds.Username == "" || creds.Secret == "" {
		return nil, fmt.Errorf("凭证信息不完整")
	}
	
	return &Credentials{
		Username: creds.Username,
		Password: creds.Secret,
	}, nil
}

// getCredentialsFromDockerConfig 从Docker配置文件读取凭证
func (c *Client) getCredentialsFromDockerConfig() (*Credentials, error) {
	// 尝试从不同的Docker配置位置读取
	configPaths := []string{
		os.Getenv("HOME") + "/.docker/config.json",
		"/etc/docker/config.json",
	}
	
	for _, configPath := range configPaths {
		if creds, err := c.readDockerConfig(configPath); err == nil {
			return creds, nil
		}
	}
	
	return nil, fmt.Errorf("无法从Docker配置文件读取凭证")
}

// readDockerConfig 读取Docker配置文件
func (c *Client) readDockerConfig(configPath string) (*Credentials, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	
	var config struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	// 查找对应registry的认证信息
	if auth, exists := config.Auths[c.registryURL]; exists && auth.Auth != "" {
		// 解码base64认证信息
		decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
		if err != nil {
			return nil, err
		}
		
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) == 2 {
			return &Credentials{
				Username: parts[0],
				Password: parts[1],
			}, nil
		}
	}
	
	return nil, fmt.Errorf("未找到对应的认证信息")
}

// loadLocalCredentials 从本地文件加载认证信息（向后兼容）
func (c *Client) loadLocalCredentials() error {
	configFile := os.Getenv("HOME") + "/.docker-genee/credentials.json"
	
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, &c.credentials)
}

// IsLoggedIn 检查是否已登录
func (c *Client) IsLoggedIn() bool {
	if c.credentials == nil {
		c.LoadCredentials()
	}
	return c.credentials != nil && c.credentials.Username != ""
}

// HasValidCredentials 检查是否有有效的认证信息
func (c *Client) HasValidCredentials() bool {
	// 优先尝试从Docker凭证存储获取
	if err := c.LoadCredentials(); err == nil && c.credentials != nil {
		return true
	}
	return false
}

// ListImages 获取镜像列表
func (c *Client) ListImages(platform string) ([]Image, error) {
	// 检查是否有有效的认证信息
	if !c.HasValidCredentials() {
		return nil, fmt.Errorf("未找到有效的认证信息，请先使用 'docker genee login' 登录")
	}
	
	// 确保认证信息已加载
	if c.credentials == nil {
		if err := c.LoadCredentials(); err != nil {
			return nil, fmt.Errorf("加载认证信息失败: %v", err)
		}
	}
	
	// 调用真实的registry API获取镜像列表，传入平台参数
	images, err := c.fetchImagesFromRegistry(platform)
	if err != nil {
		return nil, err
	}
	
	return images, nil
}

// fetchImagesFromRegistry 从registry API获取镜像列表
func (c *Client) fetchImagesFromRegistry(platform string) ([]Image, error) {
	// 构建API URL
	apiURL := fmt.Sprintf("https://%s/v2/_catalog", c.registryURL)
	
	// 创建请求
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	
	// 添加认证头
	auth := base64.StdEncoding.EncodeToString([]byte(c.credentials.Username + ":" + c.credentials.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	
	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败，状态码: %d", resp.StatusCode)
	}
	
	// 解析响应
	var catalog struct {
		Repositories []string `json:"repositories"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	
	fmt.Printf("找到 %d 个仓库，正在获取最新标签...\n", len(catalog.Repositories))
	
	// 获取每个仓库的最新标签
	var images []Image
	for i, repo := range catalog.Repositories {
		// 显示进度条
		progress := float64(i+1) / float64(len(catalog.Repositories))
		barWidth := 30
		filled := int(progress * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		fmt.Printf("\r进度: %s %d/%d", bar, i+1, len(catalog.Repositories))
		
		tags, err := c.getRepositoryTags(repo)
		if err != nil {
			continue
		}
		
		if len(tags) == 0 {
			continue
		}
		
		// 选择最新标签（通常是latest，如果没有则选择第一个）
		latestTag := tags[0]
		for _, tag := range tags {
			if tag == "latest" {
				latestTag = tag
				break
			}
		}
		
		// 获取镜像详情
		manifest, err := c.getImageManifest(repo, latestTag)
		if err != nil {
			continue
		}
		
		// 获取平台信息
		platforms := c.getImagePlatforms(repo, latestTag)
		
		// 如果指定了平台，检查是否匹配
		if platform != "" {
			platformMatched := false
			for _, imgPlatform := range platforms {
				if strings.Contains(strings.ToLower(imgPlatform), strings.ToLower(platform)) {
					platformMatched = true
					break
				}
			}
			if !platformMatched {
				continue // 跳过不匹配平台的镜像
			}
		}
		
		images = append(images, Image{
			Repository: repo,
			Tag:        latestTag,
			Digest:     manifest.Digest,
			Size:       formatSize(manifest.Size),
			Created:    manifest.Created,
			Platforms:  platforms,
		})
	}
	
	// 清除进度条
	fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")
	fmt.Printf("成功获取 %d 个镜像信息\n\n", len(images))
	return images, nil
}

// getRepositoryTags 获取仓库的标签列表
func (c *Client) getRepositoryTags(repository string) ([]string, error) {
	apiURL := fmt.Sprintf("https://%s/v2/%s/tags/list", c.registryURL, repository)
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	
	// 添加认证头
	auth := base64.StdEncoding.EncodeToString([]byte(c.credentials.Username + ":" + c.credentials.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取标签失败，状态码: %d", resp.StatusCode)
	}
	
	var tags struct {
		Tags []string `json:"tags"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	
	return tags.Tags, nil
}

// getImageManifest 获取镜像清单
func (c *Client) getImageManifest(repository, tag string) (*Manifest, error) {
	apiURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", c.registryURL, repository, tag)
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	
	// 添加认证头和Accept头
	auth := base64.StdEncoding.EncodeToString([]byte(c.credentials.Username + ":" + c.credentials.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取清单失败，状态码: %d", resp.StatusCode)
	}
	
	// 获取Digest头
	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		digest = "unknown"
	}
	
	// 解析清单获取大小和创建时间
	var manifest struct {
		Config struct {
			Size int64 `json:"size"`
		} `json:"config"`
		Layers []struct {
			Size int64 `json:"size"`
		} `json:"layers"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}
	
	// 计算总大小
	totalSize := manifest.Config.Size
	for _, layer := range manifest.Layers {
		totalSize += layer.Size
	}
	
	// 格式化当前时间为本地时间
	now := time.Now()
	created := now.Format("2006-01-02 15:04:05")
	
	return &Manifest{
		Digest: digest,
		Size:   totalSize,
		Created: created,
	}, nil
}

// formatSize 格式化大小
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// SearchImages 搜索镜像
func (c *Client) SearchImages(query, platform string, limit int) ([]SearchResult, error) {
	// 检查是否有有效的认证信息
	if !c.HasValidCredentials() {
		return nil, fmt.Errorf("未找到有效的认证信息，请先使用 'docker genee login' 登录")
	}
	
	// 确保认证信息已加载
	if c.credentials == nil {
		if err := c.LoadCredentials(); err != nil {
			return nil, fmt.Errorf("加载认证信息失败: %v", err)
		}
	}
	
	// 调用真实的registry API进行搜索
	return c.searchImagesFromRegistry(query, platform, limit)
}

// searchImagesFromRegistry 从registry API搜索镜像
func (c *Client) searchImagesFromRegistry(query, platform string, limit int) ([]SearchResult, error) {
	// 首先获取所有仓库
	apiURL := fmt.Sprintf("https://%s/v2/_catalog", c.registryURL)
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	
	// 添加认证头
	auth := base64.StdEncoding.EncodeToString([]byte(c.credentials.Username + ":" + c.credentials.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败，状态码: %d", resp.StatusCode)
	}
	
	// 解析响应
	var catalog struct {
		Repositories []string `json:"repositories"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	
	// 过滤匹配的仓库和标签
	var matchedRepos []string
	var tagPatterns []string
	for _, repo := range catalog.Repositories {
		matched, tagPattern := matchesQuery(repo, query)
		if matched {
			matchedRepos = append(matchedRepos, repo)
			if len(tagPattern) > 0 {
				tagPatterns = append(tagPatterns, tagPattern[0])
			} else {
				tagPatterns = append(tagPatterns, "")
			}
		}
	}
	
	// 限制结果数量
	if len(matchedRepos) > limit {
		matchedRepos = matchedRepos[:limit]
	}
	
	// 构建搜索结果
	var results []SearchResult
	for i, repo := range matchedRepos {
		// 获取仓库信息，传入标签模式进行匹配
		repoInfo, err := c.getRepositoryInfo(repo, tagPatterns[i])
		if err != nil {
			continue
		}
		
		results = append(results, *repoInfo)
	}
	
	return results, nil
}

// matchesQuery 检查仓库名和标签是否匹配查询
func matchesQuery(repository, query string) (bool, []string) {
	// 检查是否包含冒号分隔符（格式：repository:tag_pattern）
	if strings.Contains(query, ":") {
		parts := strings.SplitN(query, ":", 2)
		if len(parts) == 2 {
			repoPattern := parts[0]
			tagPattern := parts[1]
			
			// 调试信息（可选）
			// if strings.Contains(repository, "node") {
			// 	fmt.Printf("DEBUG: checking repo '%s' against pattern '%s'\n", repository, repoPattern)
			// }
			
			// 检查仓库名是否匹配
			repoMatched := false
			if strings.Contains(repoPattern, "*") {
				pattern := strings.ReplaceAll(repoPattern, "*", ".*")
				repoMatched, _ = regexp.MatchString(pattern, repository)
			} else {
				repoMatched = strings.Contains(strings.ToLower(repository), strings.ToLower(repoPattern))
			}
			
			if !repoMatched {
				return false, nil
			}
			
			// 如果仓库匹配，返回true，标签模式将在后续处理中匹配
			return true, []string{tagPattern}
		}
	}
	
	// 没有冒号分隔符，只匹配仓库名
	if strings.Contains(query, "*") {
		pattern := strings.ReplaceAll(query, "*", ".*")
		matched, _ := regexp.MatchString(pattern, repository)
		return matched, nil
	}
	
	// 精确匹配仓库名
	return repository == query, nil
}

// getRepositoryInfo 获取仓库信息
func (c *Client) getRepositoryInfo(repository, tagPattern string) (*SearchResult, error) {
	// 获取标签列表
	tags, err := c.getRepositoryTags(repository)
	if err != nil {
		return nil, err
	}
	
	// 获取平台信息
	platforms := c.getRepositoryPlatforms(repository, tags)
	
	// 计算仓库总大小和获取最新标签的详情
	totalSize := int64(0)
	var latestDigest, latestCreated string
	var latestTag string
	var matchedTags []string
	
	// 如果有标签模式，查找匹配的标签
	if tagPattern != "" {
		for _, tag := range tags {
			matched := matchesTagPattern(tag, tagPattern)
			if matched {
				matchedTags = append(matchedTags, tag)
			}
		}
	}
	
	// 获取最新标签（通常是latest，如果没有则选择第一个）
	latestTag = tags[0]
	for _, tag := range tags {
		if tag == "latest" {
			latestTag = tag
			break
		}
	}
	
	// 获取标签详情
	for i, tag := range tags {
		if i >= 3 { // 只检查前3个标签来计算大小
			break
		}
		manifest, err := c.getImageManifest(repository, tag)
		if err == nil {
			totalSize += manifest.Size
			// 记录最新标签的digest和created
			if tag == latestTag {
				latestDigest = manifest.Digest
				latestCreated = manifest.Created
			}
		}
	}
	
	// 如果有标签模式但没有匹配的标签，返回错误（这样该仓库就不会被包含在结果中）
	if tagPattern != "" && len(matchedTags) == 0 {
		return nil, fmt.Errorf("没有匹配的标签")
	}
	
	return &SearchResult{
		Name:        repository,
		Description: fmt.Sprintf("包含 %d 个标签", len(tags)),
		Tags:        len(tags),
		Size:        formatSize(totalSize),
		Platforms:   platforms,
		Digest:      latestDigest,
		Created:     latestCreated,
		LatestTag:   latestTag,
		MatchedTags: matchedTags,
	}, nil
}

// getImagePlatforms 获取单个镜像的平台信息
func (c *Client) getImagePlatforms(repository, tag string) []string {
	// 检查认证信息
	if c.credentials == nil {
		return []string{"unknown"}
	}
	
	// 解析manifest获取平台信息
	// 这里需要获取manifest的详细内容来解析平台信息
	apiURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", c.registryURL, repository, tag)
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return []string{"unknown"}
	}
	
	// 添加认证头
	auth := base64.StdEncoding.EncodeToString([]byte(c.credentials.Username + ":" + c.credentials.Password))
	req.Header.Set("Authorization", "Basic "+auth)
	// 优先请求manifest list（多架构），如果没有则回退到单架构manifest
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.v1+prettyjws")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return []string{"unknown"}
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return []string{"unknown"}
	}
	
	// 解析manifest
	var manifestData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&manifestData); err != nil {
		return []string{"unknown"}
	}
	
	// 检查是否是manifest list（多平台）
	if mediaType, ok := manifestData["mediaType"].(string); ok {
			// 调试信息（可选，用于开发时查看）
	// if strings.Contains(repository, "nginx") || strings.Contains(repository, "node") {
	// 	fmt.Printf("DEBUG: %s:%s mediaType = %s\n", repository, tag, mediaType)
	// }
		
		if mediaType == "application/vnd.docker.distribution.manifest.list.v2+json" {
			// 多平台manifest
			if manifests, ok := manifestData["manifests"].([]interface{}); ok {
				var platforms []string
				for _, m := range manifests {
					if manifest, ok := m.(map[string]interface{}); ok {
										// 调试信息（可选）
				// if strings.Contains(repository, "nginx") || strings.Contains(repository, "node") {
				// 	fmt.Printf("DEBUG: %s:%s manifest[%d] keys: %v\n", repository, tag, i, getMapKeys(manifest))
				// }
						if platform, ok := manifest["platform"].(map[string]interface{}); ok {
							// 调试信息（可选）
							// if strings.Contains(repository, "nginx") || strings.Contains(repository, "node") {
							// 	fmt.Printf("DEBUG: %s:%s manifest[%d] platform keys: %v\n", repository, tag, i, getMapKeys(platform))
							// }
							if os, ok := platform["os"].(string); ok {
								if arch, ok := platform["architecture"].(string); ok {
									platforms = append(platforms, fmt.Sprintf("%s/%s", os, arch))
								}
							}
						}
					}
				}
				if len(platforms) > 0 {
					return platforms
				}
			}
		} else {
			// 单平台manifest，尝试从config中获取平台信息
			if config, ok := manifestData["config"].(map[string]interface{}); ok {
				if platform, ok := config["platform"].(map[string]interface{}); ok {
												if os, ok := platform["os"].(string); ok {
								if arch, ok := platform["architecture"].(string); ok {
									platformStr := fmt.Sprintf("%s/%s", os, arch)
									return []string{platformStr}
								}
							}
				}
			}
		}
	}
	
	// 如果无法获取平台信息，返回通用平台
	return []string{"linux/amd64"}
}

// matchesTagPattern 检查标签是否匹配模式
func matchesTagPattern(tag, pattern string) bool {
	if pattern == "" {
		return false
	}
	
	// 处理通配符模式
	if strings.Contains(pattern, "*") {
		// 将通配符模式转换为正则表达式
		// *22-alpine -> .*22-alpine$ (以22-alpine结尾)
		// 22*alpine -> ^22.*alpine$ (以22开头，以alpine结尾)
		// 22-alpine* -> ^22-alpine.* (以22-alpine开头)
		// *2* -> .*2.* (包含2)
		
		var regexPattern string
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
			// *xxx* -> .*xxx.* (包含xxx)
			middle := pattern[1 : len(pattern)-1]
			regexPattern = ".*" + regexp.QuoteMeta(middle) + ".*"
		} else if strings.HasPrefix(pattern, "*") {
			// *xxx -> .*xxx$ (以xxx结尾)
			suffix := pattern[1:]
			regexPattern = ".*" + regexp.QuoteMeta(suffix) + "$"
		} else if strings.HasSuffix(pattern, "*") {
			// xxx* -> ^xxx.* (以xxx开头)
			prefix := pattern[:len(pattern)-1]
			regexPattern = "^" + regexp.QuoteMeta(prefix) + ".*"
		} else {
			// xxx*yyy -> ^xxx.*yyy$ (以xxx开头，以yyy结尾)
			parts := strings.SplitN(pattern, "*", 2)
			if len(parts) == 2 {
				prefix := parts[0]
				suffix := parts[1]
				regexPattern = "^" + regexp.QuoteMeta(prefix) + ".*" + regexp.QuoteMeta(suffix) + "$"
			} else {
				// 如果分割失败，回退到简单替换
				regexPattern = strings.ReplaceAll(pattern, "*", ".*")
			}
		}
		
		// 调试信息（可选）
		// if strings.Contains(tag, "22") || strings.Contains(tag, "2.0") {
		// 	fmt.Printf("DEBUG: tag '%s' pattern '%s' -> regex '%s'\n", tag, pattern, regexPattern)
		// }
		
		// 编译正则表达式
		re, err := regexp.Compile(regexPattern)
		if err != nil {
			// 如果正则表达式编译失败，使用简单的字符串包含
			return strings.Contains(strings.ToLower(tag), strings.ToLower(pattern))
		}
		
		// 使用正则表达式匹配
		return re.MatchString(tag)
	}
	
	// 没有通配符，使用精确匹配
	return tag == pattern
}

// getMapKeys 获取map的所有键
func getMapKeys(m map[string]interface{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// getRepositoryPlatforms 获取仓库支持的平台
func (c *Client) getRepositoryPlatforms(repository string, tags []string) []string {
	var platforms []string
	platformSet := make(map[string]bool)
	
	// 检查前几个标签的平台信息
	for i, tag := range tags {
		if i >= 5 { // 限制检查的标签数量
			break
		}
		
		_, err := c.getImageManifest(repository, tag)
		if err != nil {
			continue
		}
		
		// 这里可以解析manifest获取平台信息
		// 简化处理，返回通用平台
		platformSet["linux/amd64"] = true
		platformSet["linux/arm64"] = true
	}
	
	// 转换为切片
	for platform := range platformSet {
		platforms = append(platforms, platform)
	}
	
	if len(platforms) == 0 {
		platforms = []string{"unknown"}
	}
	
	return platforms
}

// filterImagesByPlatform 根据平台过滤镜像
func (c *Client) filterImagesByPlatform(images []Image, platform string) []Image {
	var filtered []Image
	
	for _, img := range images {
		// 检查镜像是否支持指定平台
		for _, imgPlatform := range img.Platforms {
			if strings.Contains(strings.ToLower(imgPlatform), strings.ToLower(platform)) {
				filtered = append(filtered, img)
				break
			}
		}
	}
	
	return filtered
}