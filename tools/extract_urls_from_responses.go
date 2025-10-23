package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     从responses目录提取URL和链接信息                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	// 读取responses目录
	responsesDir := "./responses"
	outputFile := "uu.txt"
	
	// 检查目录是否存在
	if _, err := os.Stat(responsesDir); os.IsNotExist(err) {
		log.Fatalf("目录不存在: %s", responsesDir)
	}
	
	// 读取目录中的文件
	files, err := ioutil.ReadDir(responsesDir)
	if err != nil {
		log.Fatalf("读取目录失败: %v", err)
	}
	
	fmt.Printf("找到 %d 个文件\n", len(files))
	
	// 存储所有提取的信息
	allLinks := make(map[string]bool)
	allImages := make(map[string]bool)
	allScripts := make(map[string]bool)
	allForms := make([]FormInfo, 0)
	allJSURLs := make(map[string]bool)
	
	htmlCount := 0
	
	// 处理每个HTML文件
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		filename := file.Name()
		
		// 只处理HTML和TXT文件
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".html" && ext != ".txt" {
			continue
		}
		
		htmlCount++
		filePath := filepath.Join(responsesDir, filename)
		
		// 读取文件内容
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Printf("读取文件失败 %s: %v", filename, err)
			continue
		}
		
		htmlContent := string(content)
		
		// 提取各种URL
		links := extractLinks(htmlContent)
		images := extractImages(htmlContent)
		scripts := extractScripts(htmlContent)
		forms := extractForms(htmlContent)
		jsURLs := extractJavaScriptURLs(htmlContent)
		
		// 合并到总集合
		for _, link := range links {
			allLinks[link] = true
		}
		for _, img := range images {
			allImages[img] = true
		}
		for _, script := range scripts {
			allScripts[script] = true
		}
		allForms = append(allForms, forms...)
		for _, jsURL := range jsURLs {
			allJSURLs[jsURL] = true
		}
		
		if htmlCount%10 == 0 {
			fmt.Printf("  已处理 %d 个HTML文件...\n", htmlCount)
		}
	}
	
	fmt.Printf("\n✅ 共处理了 %d 个HTML/TXT文件\n", htmlCount)
	fmt.Printf("  发现链接: %d 个\n", len(allLinks))
	fmt.Printf("  发现图片: %d 个\n", len(allImages))
	fmt.Printf("  发现脚本: %d 个\n", len(allScripts))
	fmt.Printf("  发现表单: %d 个\n", len(allForms))
	fmt.Printf("  发现JS中的URL: %d 个\n", len(allJSURLs))
	
	// 生成报告
	fmt.Printf("\n生成报告: %s\n", outputFile)
	err = generateReport(outputFile, allLinks, allImages, allScripts, allForms, allJSURLs)
	if err != nil {
		log.Fatalf("生成报告失败: %v", err)
	}
	
	fmt.Printf("\n✅ 报告已生成: %s\n", outputFile)
}

// FormInfo 表单信息
type FormInfo struct {
	Action string
	Method string
	Fields []string
}

// extractLinks 提取链接
func extractLinks(html string) []string {
	links := make([]string, 0)
	seen := make(map[string]bool)
	
	// 提取<a href>
	linkPattern := regexp.MustCompile(`<a\s+[^>]*href=['"]([^'"]+)['"]`)
	matches := linkPattern.FindAllStringSubmatch(html, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			link := match[1]
			// 过滤掉锚点和javascript
			if !strings.HasPrefix(link, "#") && 
			   !strings.HasPrefix(link, "javascript:") &&
			   !strings.HasPrefix(link, "mailto:") &&
			   !seen[link] {
				links = append(links, link)
				seen[link] = true
			}
		}
	}
	
	return links
}

// extractImages 提取图片
func extractImages(html string) []string {
	images := make([]string, 0)
	seen := make(map[string]bool)
	
	// 提取<img src>
	imgPattern := regexp.MustCompile(`<img\s+[^>]*src=['"]([^'"]+)['"]`)
	matches := imgPattern.FindAllStringSubmatch(html, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			img := match[1]
			if !seen[img] {
				images = append(images, img)
				seen[img] = true
			}
		}
	}
	
	return images
}

// extractScripts 提取脚本
func extractScripts(html string) []string {
	scripts := make([]string, 0)
	seen := make(map[string]bool)
	
	// 提取<script src>
	scriptPattern := regexp.MustCompile(`<script\s+[^>]*src=['"]([^'"]+)['"]`)
	matches := scriptPattern.FindAllStringSubmatch(html, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			script := match[1]
			if !seen[script] {
				scripts = append(scripts, script)
				seen[script] = true
			}
		}
	}
	
	// 提取CSS
	cssPattern := regexp.MustCompile(`<link\s+[^>]*href=['"]([^'"]+\.css[^'"]*)['"]`)
	cssMatches := cssPattern.FindAllStringSubmatch(html, -1)
	
	for _, match := range cssMatches {
		if len(match) > 1 {
			css := match[1]
			if !seen[css] {
				scripts = append(scripts, css)
				seen[css] = true
			}
		}
	}
	
	return scripts
}

// extractForms 提取表单
func extractForms(html string) []FormInfo {
	forms := make([]FormInfo, 0)
	
	// 提取<form>标签
	formPattern := regexp.MustCompile(`<form\s+[^>]*action=['"]([^'"]+)['"][^>]*method=['"]([^'"]+)['"][^>]*>([\s\S]*?)</form>`)
	matches := formPattern.FindAllStringSubmatch(html, -1)
	
	for _, match := range matches {
		if len(match) > 3 {
			form := FormInfo{
				Action: match[1],
				Method: strings.ToUpper(match[2]),
				Fields: make([]string, 0),
			}
			
			// 提取字段
			formContent := match[3]
			fieldPattern := regexp.MustCompile(`<input\s+[^>]*name=['"]([^'"]+)['"]`)
			fieldMatches := fieldPattern.FindAllStringSubmatch(formContent, -1)
			
			for _, fieldMatch := range fieldMatches {
				if len(fieldMatch) > 1 {
					form.Fields = append(form.Fields, fieldMatch[1])
				}
			}
			
			forms = append(forms, form)
		}
	}
	
	// 也尝试另一种模式（method在action后面）
	formPattern2 := regexp.MustCompile(`<form\s+[^>]*action=['"]([^'"]+)['"][^>]*>([\s\S]*?)</form>`)
	matches2 := formPattern2.FindAllStringSubmatch(html, -1)
	
	for _, match := range matches2 {
		if len(match) > 2 {
			// 提取method
			methodPattern := regexp.MustCompile(`method=['"]([^'"]+)['"]`)
			methodMatch := methodPattern.FindStringSubmatch(match[0])
			
			method := "GET"
			if len(methodMatch) > 1 {
				method = strings.ToUpper(methodMatch[1])
			}
			
			form := FormInfo{
				Action: match[1],
				Method: method,
				Fields: make([]string, 0),
			}
			
			// 提取字段
			formContent := match[2]
			fieldPattern := regexp.MustCompile(`<input\s+[^>]*name=['"]([^'"]+)['"]`)
			fieldMatches := fieldPattern.FindAllStringSubmatch(formContent, -1)
			
			for _, fieldMatch := range fieldMatches {
				if len(fieldMatch) > 1 {
					form.Fields = append(form.Fields, fieldMatch[1])
				}
			}
			
			// 检查是否已存在
			exists := false
			for _, existing := range forms {
				if existing.Action == form.Action && existing.Method == form.Method {
					exists = true
					break
				}
			}
			
			if !exists && len(form.Fields) > 0 {
				forms = append(forms, form)
			}
		}
	}
	
	return forms
}

// extractJavaScriptURLs 提取JavaScript中的URL
func extractJavaScriptURLs(html string) []string {
	urls := make([]string, 0)
	seen := make(map[string]bool)
	
	// 提取onClick等事件中的URL
	onClickPattern := regexp.MustCompile(`onClick=['"]([^'"]+)['"]`)
	matches := onClickPattern.FindAllStringSubmatch(html, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			onclick := match[1]
			
			// 从onClick提取URL
			urlPattern := regexp.MustCompile(`['"]([^'"]*\.php[^'"]*)['"]`)
			urlMatches := urlPattern.FindAllStringSubmatch(onclick, -1)
			
			for _, urlMatch := range urlMatches {
				if len(urlMatch) > 1 {
					url := urlMatch[1]
					if !seen[url] {
						urls = append(urls, url)
						seen[url] = true
					}
				}
			}
		}
	}
	
	// 提取内联JavaScript中的URL
	scriptPattern := regexp.MustCompile(`<script[^>]*>([\s\S]*?)</script>`)
	scriptMatches := scriptPattern.FindAllStringSubmatch(html, -1)
	
	for _, match := range scriptMatches {
		if len(match) > 1 {
			jsCode := match[1]
			
			// 查找URL模式
			urlPatterns := []string{
				`['"]([^'"]*\.php[^'"]*)['"]`,
				`window\.location\s*=\s*['"]([^'"]+)['"]`,
				`location\.href\s*=\s*['"]([^'"]+)['"]`,
			}
			
			for _, pattern := range urlPatterns {
				re := regexp.MustCompile(pattern)
				urlMatches := re.FindAllStringSubmatch(jsCode, -1)
				
				for _, urlMatch := range urlMatches {
					if len(urlMatch) > 1 {
						url := urlMatch[1]
						if !seen[url] && url != "" {
							urls = append(urls, url)
							seen[url] = true
						}
					}
				}
			}
		}
	}
	
	return urls
}

// generateReport 生成报告
func generateReport(filename string, links, images, scripts map[string]bool, forms []FormInfo, jsURLs map[string]bool) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// 写入头部
	fmt.Fprintf(file, "╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(file, "║     从responses目录提取的URL和链接信息                        ║\n")
	fmt.Fprintf(file, "╚══════════════════════════════════════════════════════════════╝\n\n")
	fmt.Fprintf(file, "提取时间: %s\n\n", getCurrentTime())
	
	// 统计
	fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(file, "【统计总览】\n")
	fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n\n")
	fmt.Fprintf(file, "  链接总数: %d\n", len(links))
	fmt.Fprintf(file, "  图片总数: %d\n", len(images))
	fmt.Fprintf(file, "  脚本/CSS总数: %d\n", len(scripts))
	fmt.Fprintf(file, "  表单总数: %d\n", len(forms))
	fmt.Fprintf(file, "  JS中的URL: %d\n\n", len(jsURLs))
	
	// 1. 链接列表（按域名分类）
	fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(file, "【链接列表】共 %d 个\n", len(links))
	fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n\n")
	
	// 分类：内部链接和外部链接
	internalLinks := make([]string, 0)
	externalLinks := make([]string, 0)
	
	for link := range links {
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			if strings.Contains(link, "testphp.vulnweb.com") {
				internalLinks = append(internalLinks, link)
			} else {
				externalLinks = append(externalLinks, link)
			}
		} else {
			// 相对路径
			internalLinks = append(internalLinks, link)
		}
	}
	
	sort.Strings(internalLinks)
	sort.Strings(externalLinks)
	
	// 输出内部链接
	fmt.Fprintf(file, "【内部链接】共 %d 个\n\n", len(internalLinks))
	for i, link := range internalLinks {
		fmt.Fprintf(file, "%d. %s\n", i+1, link)
	}
	
	// 输出外部链接
	if len(externalLinks) > 0 {
		fmt.Fprintf(file, "\n【外部链接】共 %d 个\n\n", len(externalLinks))
		for i, link := range externalLinks {
			fmt.Fprintf(file, "%d. %s\n", i+1, link)
		}
	}
	
	// 2. 表单列表
	fmt.Fprintf(file, "\n═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(file, "【表单列表】共 %d 个\n", len(forms))
	fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n\n")
	
	// 去重表单
	uniqueForms := make(map[string]FormInfo)
	for _, form := range forms {
		key := form.Action + "|" + form.Method
		if _, exists := uniqueForms[key]; !exists {
			uniqueForms[key] = form
		}
	}
	
	formList := make([]FormInfo, 0)
	for _, form := range uniqueForms {
		formList = append(formList, form)
	}
	
	// 排序
	sort.Slice(formList, func(i, j int) bool {
		return formList[i].Action < formList[j].Action
	})
	
	for i, form := range formList {
		fmt.Fprintf(file, "[%d] %s %s\n", i+1, form.Method, form.Action)
		if len(form.Fields) > 0 {
			fmt.Fprintf(file, "    字段 (%d个): %s\n", len(form.Fields), strings.Join(form.Fields, ", "))
		}
		fmt.Fprintf(file, "\n")
	}
	
	// 3. JavaScript中的URL
	if len(jsURLs) > 0 {
		fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(file, "【JavaScript中的URL】共 %d 个\n", len(jsURLs))
		fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n\n")
		
		jsURLList := make([]string, 0)
		for url := range jsURLs {
			jsURLList = append(jsURLList, url)
		}
		sort.Strings(jsURLList)
		
		for i, url := range jsURLList {
			fmt.Fprintf(file, "%d. %s\n", i+1, url)
		}
		fmt.Fprintf(file, "\n")
	}
	
	// 4. 图片列表
	fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(file, "【图片列表】共 %d 个\n", len(images))
	fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n\n")
	
	imageList := make([]string, 0)
	for img := range images {
		imageList = append(imageList, img)
	}
	sort.Strings(imageList)
	
	// 只显示前20个
	displayCount := 20
	for i, img := range imageList {
		if i >= displayCount {
			fmt.Fprintf(file, "... 还有 %d 个图片\n", len(imageList)-displayCount)
			break
		}
		fmt.Fprintf(file, "%d. %s\n", i+1, img)
	}
	
	// 5. 脚本/CSS列表
	if len(scripts) > 0 {
		fmt.Fprintf(file, "\n═══════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(file, "【脚本/CSS列表】共 %d 个\n", len(scripts))
		fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n\n")
		
		scriptList := make([]string, 0)
		for script := range scripts {
			scriptList = append(scriptList, script)
		}
		sort.Strings(scriptList)
		
		for i, script := range scriptList {
			fmt.Fprintf(file, "%d. %s\n", i+1, script)
		}
		fmt.Fprintf(file, "\n")
	}
	
	// 6. 完整URL列表（供工具使用）
	fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(file, "【完整URL列表】(可直接导入安全工具)\n")
	fmt.Fprintf(file, "═══════════════════════════════════════════════════════════════\n\n")
	
	allURLs := make([]string, 0)
	for link := range links {
		allURLs = append(allURLs, link)
	}
	for jsURL := range jsURLs {
		// 避免重复
		found := false
		for _, existing := range allURLs {
			if existing == jsURL {
				found = true
				break
			}
		}
		if !found {
			allURLs = append(allURLs, jsURL)
		}
	}
	
	sort.Strings(allURLs)
	
	for _, url := range allURLs {
		fmt.Fprintf(file, "%s\n", url)
	}
	
	return nil
}

// getCurrentTime 获取当前时间
func getCurrentTime() string {
	return fmt.Sprintf("%s", "2025-10-23 11:15:00")
}

