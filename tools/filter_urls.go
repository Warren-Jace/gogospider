package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// 快速URL过滤工具
// 用于过滤现有爬取结果中的无效URL

func main() {
	inputFile := flag.String("input", "", "输入文件路径")
	outputFile := flag.String("output", "", "输出文件路径（可选，默认为输入文件名+_filtered）")
	showStats := flag.Bool("stats", true, "显示统计信息")
	verbose := flag.Bool("v", false, "显示详细信息（显示被过滤的URL）")
	
	flag.Parse()
	
	if *inputFile == "" {
		fmt.Println("用法: filter_urls -input <文件路径> [-output <输出路径>] [-stats] [-v]")
		fmt.Println("\n示例:")
		fmt.Println("  filter_urls -input spider_x.lydaas.com_20251026_211654_all_urls.txt")
		fmt.Println("  filter_urls -input all_urls.txt -output filtered_urls.txt -v")
		os.Exit(1)
	}
	
	// 确定输出文件名
	output := *outputFile
	if output == "" {
		ext := filepath.Ext(*inputFile)
		base := strings.TrimSuffix(*inputFile, ext)
		output = base + "_filtered" + ext
	}
	
	// 读取输入文件
	fmt.Printf("正在读取: %s\n", *inputFile)
	urls, err := readURLs(*inputFile)
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}
	
	fmt.Printf("原始URL数量: %d\n", len(urls))
	
	// 过滤URL
	fmt.Println("正在过滤...")
	filtered, removed := filterURLs(urls, *verbose)
	
	// 保存结果
	fmt.Printf("正在保存到: %s\n", output)
	err = saveURLs(output, filtered)
	if err != nil {
		log.Fatalf("保存文件失败: %v", err)
	}
	
	// 显示统计
	if *showStats {
		printStats(len(urls), len(filtered), len(removed), removed)
	}
	
	fmt.Printf("\n✅ 完成！过滤后URL数量: %d\n", len(filtered))
	fmt.Printf("   过滤率: %.1f%%\n", float64(len(removed))/float64(len(urls))*100)
}

func readURLs(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	urls := make([]string, 0)
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	
	return urls, scanner.Err()
}

func saveURLs(filename string, urls []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	
	for _, url := range urls {
		writer.WriteString(url + "\n")
	}
	
	return nil
}

func filterURLs(urls []string, verbose bool) ([]string, []string) {
	filtered := make([]string, 0)
	removed := make([]string, 0)
	
	for _, url := range urls {
		if isValidBusinessURL(url) {
			filtered = append(filtered, url)
		} else {
			removed = append(removed, url)
			if verbose {
				reason := getFilterReason(url)
				fmt.Printf("  [过滤] %s - %s\n", url, reason)
			}
		}
	}
	
	return filtered, removed
}

func isValidBusinessURL(url string) bool {
	// 基本检查
	if url == "" || url == "/" {
		return false
	}
	
	// 提取路径部分
	path := extractPath(url)
	
	// 1. 过滤URL编码的代码（超过10%是编码字符）
	encodedCount := strings.Count(url, "%")
	if encodedCount > len(url)/10 {
		return false
	}
	
	// 2. 过滤HTML标签
	if strings.Contains(url, "<") || strings.Contains(url, ">") {
		return false
	}
	
	// 3. 过滤MIME类型
	if isMIMEType(path) {
		return false
	}
	
	// 4. 过滤JavaScript关键字
	if isJSKeyword(path) {
		return false
	}
	
	// 5. 过滤太短的路径
	cleanPath := strings.Trim(path, "/")
	if len(cleanPath) > 0 && len(cleanPath) < 3 {
		// 允许一些常见的短路径
		allowed := map[string]bool{
			"ui": true, "id": true, "no": true,
			"en": true, "zh": true, "cn": true,
			"v1": true, "v2": true, "v3": true,
			"f": true,
		}
		if !allowed[strings.ToLower(cleanPath)] {
			return false
		}
	}
	
	// 6. 过滤纯数字路径
	if len(cleanPath) > 0 && len(cleanPath) < 4 {
		isNumber := true
		for _, c := range cleanPath {
			if c < '0' || c > '9' {
				isNumber = false
				break
			}
		}
		if isNumber {
			return false
		}
	}
	
	// 7. 过滤特殊字符过多
	specialCount := strings.Count(path, "{") + strings.Count(path, "}") +
		strings.Count(path, "[") + strings.Count(path, "]") +
		strings.Count(path, "(") + strings.Count(path, ")")
	if specialCount > 3 {
		return false
	}
	
	// 8. 检查是否包含代码模式
	if containsCodePattern(url) {
		return false
	}
	
	// 9. 检查路径长度
	if len(path) > 200 {
		return false
	}
	
	// 10. 检查是否有意义
	if !hasMeaningfulPath(path) {
		return false
	}
	
	return true
}

func extractPath(url string) string {
	// 移除协议
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	
	// 找到第一个/后的部分
	parts := strings.SplitN(url, "/", 2)
	if len(parts) < 2 {
		return "/"
	}
	
	// 移除查询参数
	path := "/" + parts[1]
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}
	
	return path
}

func isMIMEType(path string) bool {
	cleanPath := strings.TrimPrefix(path, "/")
	
	mimeTypes := []string{
		"application/", "text/", "image/", "video/", "audio/",
		"vnd.ms-", "vnd.openxmlformats", "x-www-form-urlencoded",
	}
	
	for _, mime := range mimeTypes {
		if strings.Contains(cleanPath, mime) {
			return true
		}
	}
	
	return false
}

func isJSKeyword(path string) bool {
	cleanPath := strings.Trim(path, "/")
	cleanPath = strings.ToLower(cleanPath)
	
	// 提取最后一段
	segments := strings.Split(cleanPath, "/")
	lastSegment := segments[len(segments)-1]
	
	keywords := map[string]bool{
		"math": true, "date": true, "array": true, "object": true,
		"string": true, "number": true, "boolean": true, "json": true,
		"codeMirror": true, "treenode": true, "workbook": true,
		"book": true, "each": true, "map": true, "filter": true,
		"reduce": true, "foreach": true, "match": true, "replace": true,
		"block": true, "inline": true, "none": true, "flex": true,
		"div": true, "span": true, "table": true, "form": true,
		"input": true, "button": true, "a": true, "b": true,
		"i": true, "p": true, "h": true, "d": true, "e": true,
		"f": false, // f 可能是有效路径
		"g": true, "m": true, "n": true, "o": true, "r": true,
		"t": true, "y": true, "can": true, "has": true, "is": true,
		"get": true, "set": true, "add": true, "del": true,
		"new": true, "pro": true, "sub": true, "sup": true,
	}
	
	// 检查完整路径
	if keywords[cleanPath] {
		return true
	}
	
	// 检查最后一段
	if keywords[lastSegment] {
		return true
	}
	
	return false
}

func containsCodePattern(url string) bool {
	codePatterns := []string{
		"function", "var ", "let ", "const ",
		"===", "!==", ".concat(", ".replace(", ".slice(",
		"//", "/*", "*/",
	}
	
	for _, pattern := range codePatterns {
		if strings.Contains(url, pattern) {
			return true
		}
	}
	
	return false
}

func hasMeaningfulPath(path string) bool {
	cleanPath := strings.Trim(path, "/")
	
	if cleanPath == "" {
		return true // 根路径有效
	}
	
	// 检查业务关键词
	businessKeywords := []string{
		"api", "admin", "user", "login", "logout", "register",
		"account", "profile", "setting", "config", "management",
		"list", "detail", "edit", "create", "update", "delete",
		"search", "query", "export", "import", "download", "upload",
		"home", "index", "main", "dashboard", "workbench",
		"page", "view", "portal", "center", "blank", "simple",
		"harbor", "document", "fund", "commodity", "trade",
		"integration", "property", "epoch",
	}
	
	pathLower := strings.ToLower(cleanPath)
	for _, keyword := range businessKeywords {
		if strings.Contains(pathLower, keyword) {
			return true
		}
	}
	
	// 检查文件扩展名
	exts := []string{
		".php", ".asp", ".aspx", ".jsp", ".do",
		".html", ".htm", ".json", ".xml",
	}
	
	for _, ext := range exts {
		if strings.HasSuffix(pathLower, ext) {
			return true
		}
	}
	
	// 多段路径通常有意义
	if strings.Count(cleanPath, "/") >= 1 {
		return true
	}
	
	return false
}

func getFilterReason(url string) string {
	path := extractPath(url)
	
	encodedCount := strings.Count(url, "%")
	if encodedCount > len(url)/10 {
		return "URL编码过多（可能是代码）"
	}
	
	if strings.Contains(url, "<") || strings.Contains(url, ">") {
		return "包含HTML标签"
	}
	
	if isMIMEType(path) {
		return "MIME类型"
	}
	
	if isJSKeyword(path) {
		return "JavaScript关键字"
	}
	
	cleanPath := strings.Trim(path, "/")
	if len(cleanPath) > 0 && len(cleanPath) < 3 {
		return "路径过短"
	}
	
	if containsCodePattern(url) {
		return "包含代码模式"
	}
	
	if len(path) > 200 {
		return "路径过长"
	}
	
	if !hasMeaningfulPath(path) {
		return "无业务意义"
	}
	
	return "未知原因"
}

func printStats(total, filtered, removed int, removedURLs []string) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("过滤统计")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("原始URL数: %d\n", total)
	fmt.Printf("保留URL数: %d (%.1f%%)\n", filtered, float64(filtered)/float64(total)*100)
	fmt.Printf("过滤URL数: %d (%.1f%%)\n", removed, float64(removed)/float64(total)*100)
	
	// 统计过滤原因
	reasons := make(map[string]int)
	for _, url := range removedURLs {
		reason := getFilterReason(url)
		reasons[reason]++
	}
	
	fmt.Println("\n过滤原因分布:")
	for reason, count := range reasons {
		fmt.Printf("  - %s: %d (%.1f%%)\n", reason, count, float64(count)/float64(removed)*100)
	}
	
	fmt.Println(strings.Repeat("=", 70))
}

