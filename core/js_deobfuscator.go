package core

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// JSDeobfuscator JavaScript反混淆器
type JSDeobfuscator struct {
	code          string
	originalCode  string
	decodedStrings map[string]string
	statistics    map[string]int
}

// NewJSDeobfuscator 创建JS反混淆器
func NewJSDeobfuscator(jsCode string) *JSDeobfuscator {
	return &JSDeobfuscator{
		code:          jsCode,
		originalCode:  jsCode,
		decodedStrings: make(map[string]string),
		statistics:    make(map[string]int),
	}
}

// Deobfuscate 执行反混淆
func (jd *JSDeobfuscator) Deobfuscate() string {
	fmt.Println("[JS反混淆] 开始分析...")
	
	// 1. 检测混淆类型
	obfuscationType := jd.detectObfuscationType()
	fmt.Printf("  [混淆类型] %s\n", obfuscationType)
	
	// 2. Base64解码
	jd.decodeBase64Strings()
	
	// 3. Hex解码
	jd.decodeHexStrings()
	
	// 4. Unicode解码
	jd.decodeUnicodeStrings()
	
	// 5. 字符串拼接还原
	jd.reconstructConcatenatedStrings()
	
	// 6. 数组解密
	jd.decryptArrays()
	
	// 7. 常量折叠
	jd.constantFolding()
	
	// 8. 死代码消除
	jd.deadCodeElimination()
	
	// 9. 控制流简化
	jd.simplifyControlFlow()
	
	// 10. 变量名还原
	jd.renameVariables()
	
	// 11. 格式美化
	jd.beautify()
	
	fmt.Printf("  [统计] 解密字符串: %d, 还原表达式: %d\n",
		jd.statistics["decoded_strings"],
		jd.statistics["reconstructed_expressions"])
	
	return jd.code
}

// detectObfuscationType 检测混淆类型
func (jd *JSDeobfuscator) detectObfuscationType() string {
	code := jd.code
	
	// 检测常见混淆器特征
	if strings.Contains(code, "obfuscator.io") || 
	   regexp.MustCompile(`_0x[0-9a-f]{4,6}`).MatchString(code) {
		return "obfuscator.io"
	}
	
	if strings.Contains(code, "jsfuck") || 
	   regexp.MustCompile(`^\[.*\]\[.*\]\[.*\]`).MatchString(code) {
		return "JSFuck"
	}
	
	if regexp.MustCompile(`eval\(function\(p,a,c,k,e,[dr]\)`).MatchString(code) {
		return "Packer"
	}
	
	if regexp.MustCompile(`var _0x[0-9a-f]+\s*=\s*\[`).MatchString(code) {
		return "Array-based"
	}
	
	// 检查混淆程度
	nonAscii := 0
	for _, r := range code {
		if r > 127 {
			nonAscii++
		}
	}
	
	if float64(nonAscii)/float64(len(code)) > 0.3 {
		return "Heavy Unicode"
	}
	
	// 检查Base64
	base64Pattern := regexp.MustCompile(`['"]([A-Za-z0-9+/=]{20,})['"]`)
	if len(base64Pattern.FindAllString(code, -1)) > 10 {
		return "Base64 Heavy"
	}
	
	return "Light/Unknown"
}

// decodeBase64Strings 解码Base64字符串
func (jd *JSDeobfuscator) decodeBase64Strings() {
	// 查找Base64字符串
	base64Pattern := regexp.MustCompile(`['"]([A-Za-z0-9+/=]{20,})['"]`)
	matches := base64Pattern.FindAllStringSubmatch(jd.code, -1)
	
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		encoded := match[1]
		
		// 尝试解码
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			continue
		}
		
		decodedStr := string(decoded)
		
		// 检查是否是可打印字符串
		if jd.isPrintable(decodedStr) {
			// 替换
			jd.code = strings.ReplaceAll(jd.code, match[0], fmt.Sprintf("'%s'", decodedStr))
			jd.decodedStrings[encoded] = decodedStr
			jd.statistics["decoded_strings"]++
		}
	}
}

// decodeHexStrings 解码Hex字符串
func (jd *JSDeobfuscator) decodeHexStrings() {
	// 查找Hex字符串: \x48\x65\x6c\x6c\x6f
	hexPattern := regexp.MustCompile(`\\x[0-9a-fA-F]{2}`)
	matches := hexPattern.FindAllString(jd.code, -1)
	
	if len(matches) == 0 {
		return
	}
	
	// 连续的hex序列
	hexSeqPattern := regexp.MustCompile(`['"]([\\x[0-9a-fA-F]{2}]+)['"]`)
	seqMatches := hexSeqPattern.FindAllStringSubmatch(jd.code, -1)
	
	for _, match := range seqMatches {
		if len(match) < 2 {
			continue
		}
		
		hexSeq := match[1]
		
		// 提取所有hex值
		hexes := hexPattern.FindAllString(hexSeq, -1)
		decoded := ""
		
		for _, h := range hexes {
			// 移除\x前缀
			hexStr := strings.TrimPrefix(h, "\\x")
			b, err := hex.DecodeString(hexStr)
			if err != nil {
				continue
			}
			decoded += string(b)
		}
		
		if jd.isPrintable(decoded) {
			jd.code = strings.ReplaceAll(jd.code, match[0], fmt.Sprintf("'%s'", decoded))
			jd.statistics["decoded_strings"]++
		}
	}
}

// decodeUnicodeStrings 解码Unicode字符串
func (jd *JSDeobfuscator) decodeUnicodeStrings() {
	// 查找Unicode: \u0048\u0065\u006c\u006c\u006f
	unicodePattern := regexp.MustCompile(`\\u[0-9a-fA-F]{4}`)
	
	unicodeSeqPattern := regexp.MustCompile(`['"]([\\u[0-9a-fA-F]{4}]+)['"]`)
	matches := unicodeSeqPattern.FindAllStringSubmatch(jd.code, -1)
	
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		unicodeSeq := match[1]
		unicodes := unicodePattern.FindAllString(unicodeSeq, -1)
		
		decoded := ""
		for _, u := range unicodes {
			// 移除\u前缀
			hexStr := strings.TrimPrefix(u, "\\u")
			codePoint, err := strconv.ParseInt(hexStr, 16, 32)
			if err != nil {
				continue
			}
			decoded += string(rune(codePoint))
		}
		
		if jd.isPrintable(decoded) {
			jd.code = strings.ReplaceAll(jd.code, match[0], fmt.Sprintf("'%s'", decoded))
			jd.statistics["decoded_strings"]++
		}
	}
}

// reconstructConcatenatedStrings 重建拼接的字符串
func (jd *JSDeobfuscator) reconstructConcatenatedStrings() {
	// 查找字符串拼接: 'Hello' + ' ' + 'World'
	concatPattern := regexp.MustCompile(`['"]([^'"]+)['"]\s*\+\s*['"]([^'"]+)['"]`)
	
	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		matches := concatPattern.FindAllStringSubmatch(jd.code, -1)
		if len(matches) == 0 {
			break
		}
		
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			
			// 合并字符串
			combined := match[1] + match[2]
			jd.code = strings.ReplaceAll(jd.code, match[0], fmt.Sprintf("'%s'", combined))
			jd.statistics["reconstructed_expressions"]++
		}
	}
	
	// 处理更复杂的情况: var1 + var2 + 'string'
	// 这需要变量追踪，暂时简化处理
}

// decryptArrays 解密数组
func (jd *JSDeobfuscator) decryptArrays() {
	// 查找加密数组: var _0x1234 = ['str1', 'str2', ...]
	arrayPattern := regexp.MustCompile(`var\s+(_0x[0-9a-f]+)\s*=\s*\[(.*?)\];`)
	matches := arrayPattern.FindAllStringSubmatch(jd.code, -1)
	
	arrays := make(map[string][]string)
	
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		
		varName := match[1]
		arrayContent := match[2]
		
		// 解析数组元素
		elements := jd.parseArrayElements(arrayContent)
		arrays[varName] = elements
	}
	
	// 查找数组访问: _0x1234[0]
	for varName, elements := range arrays {
		accessPattern := regexp.MustCompile(varName + `\[(\d+)\]`)
		accessMatches := accessPattern.FindAllStringSubmatch(jd.code, -1)
		
		for _, match := range accessMatches {
			if len(match) < 2 {
				continue
			}
			
			index, err := strconv.Atoi(match[1])
			if err != nil || index >= len(elements) {
				continue
			}
			
			// 替换
			jd.code = strings.ReplaceAll(jd.code, match[0], elements[index])
			jd.statistics["decoded_strings"]++
		}
	}
}

// parseArrayElements 解析数组元素
func (jd *JSDeobfuscator) parseArrayElements(content string) []string {
	elements := make([]string, 0)
	
	// 简单分割（实际应该用AST解析）
	parts := strings.Split(content, ",")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// 移除引号
		part = strings.Trim(part, "'\"")
		elements = append(elements, part)
	}
	
	return elements
}

// constantFolding 常量折叠
func (jd *JSDeobfuscator) constantFolding() {
	// 1. 算术表达式: 1 + 2 → 3
	arithPattern := regexp.MustCompile(`(\d+)\s*([+\-*/])\s*(\d+)`)
	
	for i := 0; i < 5; i++ {
		matches := arithPattern.FindAllStringSubmatch(jd.code, -1)
		if len(matches) == 0 {
			break
		}
		
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}
			
			a, _ := strconv.Atoi(match[1])
			op := match[2]
			b, _ := strconv.Atoi(match[3])
			
			var result int
			switch op {
			case "+":
				result = a + b
			case "-":
				result = a - b
			case "*":
				result = a * b
			case "/":
				if b != 0 {
					result = a / b
				}
			}
			
			jd.code = strings.ReplaceAll(jd.code, match[0], strconv.Itoa(result))
		}
	}
	
	// 2. 布尔表达式: true && false → false
	boolPatterns := map[string]string{
		`true\s*&&\s*true`:   "true",
		`true\s*&&\s*false`:  "false",
		`false\s*&&\s*true`:  "false",
		`false\s*&&\s*false`: "false",
		`true\s*\|\|\s*true`:   "true",
		`true\s*\|\|\s*false`:  "true",
		`false\s*\|\|\s*true`:  "true",
		`false\s*\|\|\s*false`: "false",
		`!true`:  "false",
		`!false`: "true",
	}
	
	for pattern, replacement := range boolPatterns {
		re := regexp.MustCompile(pattern)
		jd.code = re.ReplaceAllString(jd.code, replacement)
	}
}

// deadCodeElimination 死代码消除
func (jd *JSDeobfuscator) deadCodeElimination() {
	// 1. 移除 if (false) { ... }
	deadIfPattern := regexp.MustCompile(`if\s*\(\s*false\s*\)\s*\{[^}]*\}`)
	jd.code = deadIfPattern.ReplaceAllString(jd.code, "")
	
	// 2. 简化 if (true) { ... } → ...
	trueIfPattern := regexp.MustCompile(`if\s*\(\s*true\s*\)\s*\{([^}]*)\}`)
	matches := trueIfPattern.FindAllStringSubmatch(jd.code, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			jd.code = strings.ReplaceAll(jd.code, match[0], match[1])
		}
	}
	
	// 3. 移除空语句
	jd.code = regexp.MustCompile(`;\s*;`).ReplaceAllString(jd.code, ";")
}

// simplifyControlFlow 简化控制流
func (jd *JSDeobfuscator) simplifyControlFlow() {
	// 查找并简化switch-case混淆
	// switch (x) { case 0: ...; break; case 1: ...; break; }
	
	// 这是一个复杂的过程，需要AST分析
	// 这里只做基础简化
	
	// 移除无用的break
	jd.code = regexp.MustCompile(`break;\s*case`).ReplaceAllString(jd.code, "case")
}

// renameVariables 变量名还原
func (jd *JSDeobfuscator) renameVariables() {
	// 查找短变量名并尝试推断用途
	varPattern := regexp.MustCompile(`var\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s*=`)
	matches := varPattern.FindAllStringSubmatch(jd.code, -1)
	
	varNames := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 {
			varNames[match[1]] = true
		}
	}
	
	// 简单的启发式重命名
	for varName := range varNames {
		if len(varName) <= 2 {
			// 查找变量的使用上下文
			contextPattern := regexp.MustCompile(varName + `\s*=\s*['"]([^'"]+)['"]`)
			contextMatch := contextPattern.FindStringSubmatch(jd.code)
			
			if len(contextMatch) >= 2 {
				value := contextMatch[1]
				// 如果赋值是URL，重命名为url_xxx
				if strings.HasPrefix(value, "http") {
					newName := "url_" + varName
					jd.code = strings.ReplaceAll(jd.code, varName, newName)
				}
			}
		}
	}
}

// beautify 格式美化
func (jd *JSDeobfuscator) beautify() {
	// 1. 移除多余空白
	jd.code = regexp.MustCompile(`\s+`).ReplaceAllString(jd.code, " ")
	
	// 2. 添加换行
	jd.code = strings.ReplaceAll(jd.code, ";", ";\n")
	jd.code = strings.ReplaceAll(jd.code, "{", "{\n")
	jd.code = strings.ReplaceAll(jd.code, "}", "}\n")
	
	// 3. 基础缩进
	lines := strings.Split(jd.code, "\n")
	indentLevel := 0
	result := make([]string, 0)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// 减少缩进
		if strings.HasPrefix(line, "}") {
			indentLevel--
			if indentLevel < 0 {
				indentLevel = 0
			}
		}
		
		// 添加缩进
		result = append(result, strings.Repeat("  ", indentLevel)+line)
		
		// 增加缩进
		if strings.HasSuffix(line, "{") {
			indentLevel++
		}
	}
	
	jd.code = strings.Join(result, "\n")
}

// ExtractHiddenURLs 提取隐藏的URL
func (jd *JSDeobfuscator) ExtractHiddenURLs() []string {
	urls := make(map[string]bool)
	
	// 1. 从解码后的字符串中提取URL
	for _, decoded := range jd.decodedStrings {
		if strings.HasPrefix(decoded, "http") || strings.HasPrefix(decoded, "/") {
			urls[decoded] = true
		}
	}
	
	// 2. 查找URL模式
	urlPatterns := []string{
		// 完整URL
		`https?://[^\s'"<>]+`,
		// 路径
		`['"]/(api|v\d+|admin|user|auth|login)/[^\s'"]+['"]`,
		// 域名
		`['"]([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}['"]`,
	}
	
	for _, pattern := range urlPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(jd.code, -1)
		
		for _, match := range matches {
			// 清理引号
			match = strings.Trim(match, "'\"")
			if match != "" && len(match) > 3 {
				urls[match] = true
			}
		}
	}
	
	// 3. 查找Base64编码的URL
	base64Pattern := regexp.MustCompile(`['"]([A-Za-z0-9+/=]{30,})['"]`)
	matches := base64Pattern.FindAllStringSubmatch(jd.code, -1)
	
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		decoded, err := base64.StdEncoding.DecodeString(match[1])
		if err != nil {
			continue
		}
		
		decodedStr := string(decoded)
		if strings.HasPrefix(decodedStr, "http") || strings.HasPrefix(decodedStr, "/") {
			urls[decodedStr] = true
		}
	}
	
	// 转换为列表
	result := make([]string, 0, len(urls))
	for url := range urls {
		result = append(result, url)
	}
	
	return result
}

// ExtractAPIEndpoints 提取API端点
func (jd *JSDeobfuscator) ExtractAPIEndpoints() []string {
	endpoints := make(map[string]bool)
	
	// 1. 查找API相关模式
	apiPatterns := []string{
		`['"]/(api|API)/[a-zA-Z0-9/_-]+['"]`,
		`['"]/(v\d+)/[a-zA-Z0-9/_-]+['"]`,
		`baseURL\s*[+=]\s*['"]([^'"]+)['"]`,
		`apiUrl\s*[+=]\s*['"]([^'"]+)['"]`,
		`endpoint\s*[+=]\s*['"]([^'"]+)['"]`,
	}
	
	for _, pattern := range apiPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(jd.code, -1)
		
		for _, match := range matches {
			endpoint := ""
			if len(match) >= 2 {
				endpoint = match[1]
			} else {
				endpoint = match[0]
			}
			
			endpoint = strings.Trim(endpoint, "'\"")
			if endpoint != "" {
				endpoints[endpoint] = true
			}
		}
	}
	
	// 2. 查找fetch/ajax调用
	fetchPattern := regexp.MustCompile(`(fetch|axios\.(get|post)|jQuery\.ajax)\s*\(\s*['"]([^'"]+)['"]`)
	matches := fetchPattern.FindAllStringSubmatch(jd.code, -1)
	
	for _, match := range matches {
		if len(match) >= 4 {
			endpoint := match[3]
			if endpoint != "" {
				endpoints[endpoint] = true
			}
		}
	}
	
	// 转换为列表
	result := make([]string, 0, len(endpoints))
	for ep := range endpoints {
		result = append(result, ep)
	}
	
	return result
}

// GetStatistics 获取统计信息
func (jd *JSDeobfuscator) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"original_length":           len(jd.originalCode),
		"deobfuscated_length":       len(jd.code),
		"decoded_strings":           jd.statistics["decoded_strings"],
		"reconstructed_expressions": jd.statistics["reconstructed_expressions"],
		"compression_ratio":         float64(len(jd.code)) / float64(len(jd.originalCode)),
	}
}

// isPrintable 检查字符串是否可打印
func (jd *JSDeobfuscator) isPrintable(s string) bool {
	if len(s) == 0 {
		return false
	}
	
	printableCount := 0
	for _, r := range s {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			printableCount++
		}
	}
	
	// 至少80%是可打印字符
	return float64(printableCount)/float64(len(s)) >= 0.8
}

// SaveToFile 保存到文件
func (jd *JSDeobfuscator) SaveToFile(filename string) error {
	return ioutil.WriteFile(filename, []byte(jd.code), 0644)
}
