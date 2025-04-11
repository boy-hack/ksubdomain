package predict

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"strings"
)

//go:embed data/regular.cfg
var cfg string

//go:embed data/regular.dict
var dict string

// DomainGenerator 用于生成预测域名
type DomainGenerator struct {
	categories map[string][]string // 存储块分类和对应的值
	patterns   []string            // 域名组合模式
	subdomain  string              // 子域名部分
	domain     string              // 根域名部分
	output     io.Writer           // 输出接口
	count      int                 // 生成的域名计数
}

// NewDomainGenerator 创建一个新的域名生成器
func NewDomainGenerator(output io.Writer) (*DomainGenerator, error) {
	// 创建生成器实例
	dg := &DomainGenerator{
		categories: make(map[string][]string),
		output:     output,
	}

	// 加载分类字典
	if err := dg.loadDictionary(); err != nil {
		return nil, fmt.Errorf("加载字典文件失败: %v", err)
	}

	// 加载配置模式
	if err := dg.loadPatterns(); err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %v", err)
	}

	return dg, nil
}

// 从字典文件加载分类信息
func (dg *DomainGenerator) loadDictionary() error {
	scanner := bufio.NewScanner(strings.NewReader(dict))
	var currentCategory string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 检查是否是分类标识 [category]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentCategory = line[1 : len(line)-1]
			dg.categories[currentCategory] = []string{}
		} else if currentCategory != "" {
			// 如果有当前分类，添加值
			dg.categories[currentCategory] = append(dg.categories[currentCategory], line)
		}
	}

	return scanner.Err()
}

// 从配置文件加载域名生成模式
func (dg *DomainGenerator) loadPatterns() error {
	scanner := bufio.NewScanner(strings.NewReader(cfg))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			dg.patterns = append(dg.patterns, line)
		}
	}

	return scanner.Err()
}

// SetBaseDomain 设置基础域名
func (dg *DomainGenerator) SetBaseDomain(domain string) {
	// 分离子域名和根域名
	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		// 如果只有根域名 (example.com)
		dg.subdomain = ""
		dg.domain = domain
	} else {
		// 有子域名 (sub.example.com)
		dg.subdomain = parts[0]
		dg.domain = strings.Join(parts[1:], ".")
	}
}

// GenerateDomains 生成预测域名并实时输出
func (dg *DomainGenerator) GenerateDomains() int {
	dg.count = 0

	// 如果没有设置子域名，则直接返回
	if dg.subdomain == "" && dg.domain == "" {
		return dg.count
	}

	// 遍历所有模式
	for _, pattern := range dg.patterns {
		// 递归处理每个模式中的标签替换
		dg.processPattern(pattern, map[string]string{
			"subdomain": dg.subdomain,
			"domain":    dg.domain,
		})
	}

	return dg.count
}

// processPattern 递归处理模式中的标签替换
func (dg *DomainGenerator) processPattern(pattern string, replacements map[string]string) {
	// 查找第一个标签
	startIdx := strings.Index(pattern, "{")
	if startIdx == -1 {
		// 没有更多标签，输出最终结果
		if pattern != "" {
			fmt.Fprint(dg.output, pattern)
			dg.count++
		}
		return
	}

	endIdx := strings.Index(pattern, "}")
	if endIdx == -1 || endIdx < startIdx {
		// 标签格式不正确，直接返回
		return
	}

	// 提取标签名
	tagName := pattern[startIdx+1 : endIdx]

	// 检查是否已有替换值
	if value, exists := replacements[tagName]; exists {
		// 已有替换值，直接替换并继续处理
		newPattern := pattern[:startIdx] + value + pattern[endIdx+1:]
		dg.processPattern(newPattern, replacements)
		return
	}

	// 从分类中获取替换值
	values, exists := dg.categories[tagName]
	if !exists || len(values) == 0 {
		// 没有找到替换值，跳过此标签
		newPattern := pattern[:startIdx] + pattern[endIdx+1:]
		dg.processPattern(newPattern, replacements)
		return
	}

	// 对每个可能的替换值递归处理
	for _, value := range values {
		// 创建新的替换映射
		newReplacements := make(map[string]string)
		for k, v := range replacements {
			newReplacements[k] = v
		}
		newReplacements[tagName] = value

		// 替换当前标签并继续处理
		newPattern := pattern[:startIdx] + value + pattern[endIdx+1:]
		dg.processPattern(newPattern, newReplacements)
	}
}

// PredictDomains 根据给定域名预测可能的域名变体，直接输出结果
func PredictDomains(domain string, output io.Writer) (int, error) {
	// 创建域名生成器
	generator, err := NewDomainGenerator(output)
	if err != nil {
		return 0, err
	}

	// 设置基础域名
	generator.SetBaseDomain(domain)

	// 生成预测域名并返回生成的数量
	return generator.GenerateDomains(), nil
}
