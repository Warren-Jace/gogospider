package core

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// SmartFormFiller 智能表单填充器
type SmartFormFiller struct {
	// 字段模式映射表
	fieldPatterns map[string][]string
	
	// Fuzz测试载荷
	fuzzPayloads map[string][]string
	
	// 随机种子
	random *rand.Rand
}

// NewSmartFormFiller 创建智能表单填充器
func NewSmartFormFiller() *SmartFormFiller {
	sff := &SmartFormFiller{
		fieldPatterns: make(map[string][]string),
		fuzzPayloads:  make(map[string][]string),
		random:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	
	sff.initializePatterns()
	sff.initializeFuzzPayloads()
	
	return sff
}

// initializePatterns 初始化字段模式映射表
func (sff *SmartFormFiller) initializePatterns() {
	sff.fieldPatterns = map[string][]string{
		// === 邮箱类 ===
		"email": {
			"email", "邮箱", "mail", "e-mail", "correo", "epost",
			"user_email", "userEmail", "user_mail", "contact_email",
		},
		
		// === 用户名类 ===
		"username": {
			"username", "用户名", "user", "account", "usuario", "login",
			"user_name", "userName", "login_name", "account_name",
		},
		
		// === 密码类 ===
		"password": {
			"password", "密码", "pwd", "pass", "passwd", "passwort",
			"user_password", "userPassword", "login_password",
		},
		
		// === 确认密码 ===
		"confirm_password": {
			"confirm", "确认密码", "re-password", "repassword", "password2",
			"confirm_password", "confirmPassword", "password_confirm",
		},
		
		// === 手机号 ===
		"phone": {
			"phone", "手机", "mobile", "tel", "telephone", "telefono",
			"phone_number", "phoneNumber", "cell", "cellphone",
			"手机号", "联系电话", "contact_phone",
		},
		
		// === 姓名 ===
		"name": {
			"name", "姓名", "full_name", "fullname", "realname", "nombre",
			"user_name", "your_name", "contact_name", "真实姓名",
		},
		
		// === 地址 ===
		"address": {
			"address", "地址", "addr", "location", "direccion",
			"street", "street_address", "home_address", "详细地址",
		},
		
		// === 城市 ===
		"city": {
			"city", "城市", "town", "ciudad", "province", "省份",
		},
		
		// === 邮编 ===
		"zipcode": {
			"zip", "zipcode", "postal", "postcode", "邮编", "邮政编码",
			"zip_code", "postal_code",
		},
		
		// === 公司 ===
		"company": {
			"company", "公司", "organization", "org", "empresa",
			"company_name", "companyName", "单位",
		},
		
		// === 搜索 ===
		"search": {
			"search", "搜索", "query", "q", "keyword", "keywords",
			"search_query", "searchQuery", "buscar",
		},
		
		// === 留言/评论 ===
		"comment": {
			"comment", "评论", "message", "msg", "content", "text",
			"留言", "comments", "feedback", "description", "desc",
		},
		
		// === 验证码 ===
		"captcha": {
			"captcha", "验证码", "code", "verify", "verification",
			"verify_code", "verifyCode", "vcode",
		},
		
		// === URL ===
		"url": {
			"url", "website", "site", "link", "homepage", "web",
			"网址", "网站",
		},
		
		// === 年龄 ===
		"age": {
			"age", "年龄", "years",
		},
		
		// === 性别 ===
		"gender": {
			"gender", "性别", "sex", "sexo",
		},
		
		// === 数量 ===
		"quantity": {
			"quantity", "qty", "amount", "number", "num", "count",
			"数量", "cantidad",
		},
		
		// === 价格 ===
		"price": {
			"price", "价格", "cost", "fee", "amount", "precio",
		},
		
		// === 日期 ===
		"date": {
			"date", "日期", "time", "datetime", "fecha",
			"birth_date", "birthday", "生日",
		},
		
		// === ID/编号 ===
		"id": {
			"id", "编号", "number", "no", "code",
		},
	}
}

// initializeFuzzPayloads 初始化Fuzz测试载荷
func (sff *SmartFormFiller) initializeFuzzPayloads() {
	sff.fuzzPayloads = map[string][]string{
		// XSS测试载荷
		"xss": {
			`"><script>alert(1)</script>`,
			`'><script>alert(1)</script>`,
			`<img src=x onerror=alert(1)>`,
			`javascript:alert(1)`,
		},
		
		// SQL注入测试载荷
		"sqli": {
			`' OR '1'='1`,
			`" OR "1"="1`,
			`' OR 1=1--`,
			`admin' --`,
		},
		
		// 命令注入测试载荷
		"cmd": {
			`; ls -la`,
			`| whoami`,
			`& dir`,
			`$(id)`,
		},
		
		// 路径遍历测试载荷
		"path": {
			`../../../etc/passwd`,
			`..\..\..\..\windows\system32\drivers\etc\hosts`,
			`....//....//....//etc/passwd`,
		},
	}
}

// FillForm 智能填充表单
func (sff *SmartFormFiller) FillForm(form *Form, mode string) {
	for i := range form.Fields {
		field := &form.Fields[i]
		
		// 根据模式选择填充策略
		switch mode {
		case "normal":
			sff.fillNormalValue(field)
		case "fuzz":
			sff.fillFuzzValue(field)
		case "security":
			sff.fillSecurityTestValue(field)
		default:
			sff.fillNormalValue(field)
		}
	}
}

// fillNormalValue 填充正常值
func (sff *SmartFormFiller) fillNormalValue(field *FormField) {
	// 如果已有值且不是占位符，保留原值
	if field.Value != "" && !sff.isPlaceholder(field.Value) {
		return
	}
	
	fieldName := strings.ToLower(field.Name)
	fieldType := strings.ToLower(field.Type)
	
	// 根据字段类型填充
	switch fieldType {
	case "email":
		field.Value = "test@example.com"
		return
		
	case "password":
		if sff.matchPattern(fieldName, "confirm_password") {
			field.Value = "Test@123456"
		} else {
			field.Value = "Test@123456"
		}
		return
		
	case "tel", "phone":
		field.Value = "13800138000"
		return
		
	case "url":
		field.Value = "https://example.com"
		return
		
	case "number":
		if sff.matchPattern(fieldName, "age") {
			field.Value = "25"
		} else if sff.matchPattern(fieldName, "quantity") {
			field.Value = "1"
		} else if sff.matchPattern(fieldName, "price") {
			field.Value = "99"
		} else {
			field.Value = "100"
		}
		return
		
	case "date":
		field.Value = "2025-01-01"
		return
		
	case "checkbox":
		field.Value = "on"
		return
		
	case "radio":
		field.Value = field.Value // 保持原值
		return
		
	case "hidden":
		// 隐藏字段保持原值
		return
	}
	
	// 根据字段名智能匹配
	if sff.matchPattern(fieldName, "email") {
		field.Value = "test@example.com"
	} else if sff.matchPattern(fieldName, "username") {
		field.Value = "testuser"
	} else if sff.matchPattern(fieldName, "password") {
		field.Value = "Test@123456"
	} else if sff.matchPattern(fieldName, "phone") {
		field.Value = "13800138000"
	} else if sff.matchPattern(fieldName, "name") {
		field.Value = "张三"
	} else if sff.matchPattern(fieldName, "address") {
		field.Value = "北京市朝阳区测试路123号"
	} else if sff.matchPattern(fieldName, "city") {
		field.Value = "北京"
	} else if sff.matchPattern(fieldName, "zipcode") {
		field.Value = "100000"
	} else if sff.matchPattern(fieldName, "company") {
		field.Value = "测试公司"
	} else if sff.matchPattern(fieldName, "search") {
		field.Value = "test"
	} else if sff.matchPattern(fieldName, "comment") {
		field.Value = "这是一条测试评论"
	} else if sff.matchPattern(fieldName, "captcha") {
		field.Value = "1234"
	} else if sff.matchPattern(fieldName, "url") {
		field.Value = "https://example.com"
	} else if sff.matchPattern(fieldName, "age") {
		field.Value = "25"
	} else if sff.matchPattern(fieldName, "gender") {
		field.Value = "男"
	} else if sff.matchPattern(fieldName, "quantity") {
		field.Value = "1"
	} else if sff.matchPattern(fieldName, "price") {
		field.Value = "99"
	} else if sff.matchPattern(fieldName, "date") {
		field.Value = "2025-01-01"
	} else if sff.matchPattern(fieldName, "id") {
		field.Value = "12345"
	} else {
		// 默认值
		field.Value = "test_value"
	}
}

// fillFuzzValue 填充Fuzz测试值
func (sff *SmartFormFiller) fillFuzzValue(field *FormField) {
	fieldName := strings.ToLower(field.Name)
	
	// 根据字段类型选择合适的Fuzz载荷
	if sff.matchPattern(fieldName, "search") || sff.matchPattern(fieldName, "comment") {
		// 搜索和评论字段：XSS测试
		payloads := sff.fuzzPayloads["xss"]
		field.Value = payloads[sff.random.Intn(len(payloads))]
	} else if sff.matchPattern(fieldName, "id") || strings.Contains(fieldName, "id") {
		// ID类字段：SQL注入测试
		payloads := sff.fuzzPayloads["sqli"]
		field.Value = payloads[sff.random.Intn(len(payloads))]
	} else if sff.matchPattern(fieldName, "url") || strings.Contains(fieldName, "file") {
		// URL/文件类：路径遍历测试
		payloads := sff.fuzzPayloads["path"]
		field.Value = payloads[sff.random.Intn(len(payloads))]
	} else {
		// 默认：XSS测试
		payloads := sff.fuzzPayloads["xss"]
		field.Value = payloads[sff.random.Intn(len(payloads))]
	}
}

// fillSecurityTestValue 填充安全测试值
func (sff *SmartFormFiller) fillSecurityTestValue(field *FormField) {
	fieldName := strings.ToLower(field.Name)
	
	// 为安全测试生成特殊值
	if sff.matchPattern(fieldName, "email") {
		field.Value = `test@example.com'"><script>alert(1)</script>`
	} else if sff.matchPattern(fieldName, "username") {
		field.Value = `admin' OR '1'='1`
	} else if sff.matchPattern(fieldName, "password") {
		field.Value = `Test@123'; DROP TABLE users; --`
	} else {
		// 组合多种测试载荷
		field.Value = `test'"><script>alert(1)</script>`
	}
}

// matchPattern 匹配字段模式
func (sff *SmartFormFiller) matchPattern(fieldName string, patternKey string) bool {
	patterns, exists := sff.fieldPatterns[patternKey]
	if !exists {
		return false
	}
	
	fieldName = strings.ToLower(fieldName)
	
	for _, pattern := range patterns {
		// 使用模糊匹配
		if sff.fuzzyMatch(fieldName, pattern) {
			return true
		}
	}
	
	return false
}

// fuzzyMatch 模糊匹配函数 - 支持多种匹配策略
func (sff *SmartFormFiller) fuzzyMatch(fieldName, keyword string) bool {
	fieldName = strings.ToLower(fieldName)
	keyword = strings.ToLower(keyword)
	
	// 策略1: 完全包含匹配
	if strings.Contains(fieldName, keyword) {
		return true
	}
	
	// 策略2: 分词匹配（处理下划线和驼峰命名）
	// 例如: user_email 匹配 email, userEmail 匹配 email
	fieldParts := sff.splitFieldName(fieldName)
	for _, part := range fieldParts {
		if part == keyword {
			return true
		}
	}
	
	// 策略3: 反向包含（关键字包含字段名）
	// 例如: "mail" 可以匹配 "email"
	if strings.Contains(keyword, fieldName) && len(fieldName) >= 3 {
		return true
	}
	
	// 策略4: 编辑距离相似度匹配（针对拼写相近的情况）
	if len(fieldName) >= 3 && len(keyword) >= 3 {
		similarity := sff.calculateSimilarity(fieldName, keyword)
		if similarity >= 0.8 { // 80%相似度
			return true
		}
	}
	
	return false
}

// splitFieldName 分割字段名（支持下划线、中划线、驼峰命名）
func (sff *SmartFormFiller) splitFieldName(fieldName string) []string {
	parts := make([]string, 0)
	
	// 替换分隔符为空格
	fieldName = strings.ReplaceAll(fieldName, "_", " ")
	fieldName = strings.ReplaceAll(fieldName, "-", " ")
	fieldName = strings.ReplaceAll(fieldName, ".", " ")
	
	// 处理驼峰命名 (例如: userName -> user name)
	var result []rune
	for i, r := range fieldName {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, ' ')
		}
		result = append(result, r)
	}
	fieldName = string(result)
	
	// 分割并清理
	for _, part := range strings.Fields(fieldName) {
		part = strings.TrimSpace(strings.ToLower(part))
		if part != "" {
			parts = append(parts, part)
		}
	}
	
	return parts
}

// calculateSimilarity 计算字符串相似度（基于Levenshtein距离）
func (sff *SmartFormFiller) calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	
	len1, len2 := len(s1), len(s2)
	if len1 == 0 || len2 == 0 {
		return 0.0
	}
	
	// 计算Levenshtein距离
	distance := sff.levenshteinDistance(s1, s2)
	maxLen := len1
	if len2 > maxLen {
		maxLen = len2
	}
	
	// 转换为相似度 (1 - 距离/最大长度)
	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance 计算编辑距离
func (sff *SmartFormFiller) levenshteinDistance(s1, s2 string) int {
	len1, len2 := len(s1), len(s2)
	
	// 创建距离矩阵
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}
	
	// 动态规划计算最小编辑距离
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // 删除
				matrix[i][j-1]+1,      // 插入
				matrix[i-1][j-1]+cost, // 替换
			)
		}
	}
	
	return matrix[len1][len2]
}

// min 返回三个数中的最小值
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// isPlaceholder 判断是否为占位符
func (sff *SmartFormFiller) isPlaceholder(value string) bool {
	placeholders := []string{
		"placeholder", "请输入", "please enter", "enter your",
		"example", "示例", "hint", "提示",
	}
	
	valueLower := strings.ToLower(value)
	for _, ph := range placeholders {
		if strings.Contains(valueLower, ph) {
			return true
		}
	}
	
	return false
}

// GetFieldSuggestion 获取字段填充建议
func (sff *SmartFormFiller) GetFieldSuggestion(fieldName string, fieldType string) string {
	fieldName = strings.ToLower(fieldName)
	fieldType = strings.ToLower(fieldType)
	
	// 创建临时字段
	field := &FormField{
		Name: fieldName,
		Type: fieldType,
	}
	
	sff.fillNormalValue(field)
	
	return field.Value
}

// AddCustomPattern 添加自定义字段模式
func (sff *SmartFormFiller) AddCustomPattern(patternKey string, patterns []string) {
	sff.fieldPatterns[patternKey] = patterns
}

// AddFuzzPayload 添加自定义Fuzz载荷
func (sff *SmartFormFiller) AddFuzzPayload(category string, payloads []string) {
	sff.fuzzPayloads[category] = payloads
}

// GenerateFormVariants 生成表单变体（用于测试）
func (sff *SmartFormFiller) GenerateFormVariants(form *Form) []*Form {
	variants := make([]*Form, 0)
	
	// 正常模式
	normalForm := sff.cloneForm(form)
	sff.FillForm(normalForm, "normal")
	variants = append(variants, normalForm)
	
	// Fuzz模式
	fuzzForm := sff.cloneForm(form)
	sff.FillForm(fuzzForm, "fuzz")
	variants = append(variants, fuzzForm)
	
	// 安全测试模式
	securityForm := sff.cloneForm(form)
	sff.FillForm(securityForm, "security")
	variants = append(variants, securityForm)
	
	return variants
}

// cloneForm 克隆表单
func (sff *SmartFormFiller) cloneForm(form *Form) *Form {
	newForm := &Form{
		Action: form.Action,
		Method: form.Method,
		Fields: make([]FormField, len(form.Fields)),
	}
	
	copy(newForm.Fields, form.Fields)
	
	return newForm
}

// GetStatistics 获取填充统计信息
func (sff *SmartFormFiller) GetStatistics() map[string]int {
	stats := make(map[string]int)
	stats["total_patterns"] = len(sff.fieldPatterns)
	stats["total_fuzz_payloads"] = len(sff.fuzzPayloads)
	
	totalPayloads := 0
	for _, payloads := range sff.fuzzPayloads {
		totalPayloads += len(payloads)
	}
	stats["total_individual_payloads"] = totalPayloads
	
	return stats
}

// PrintSupportedFields 打印支持的字段类型
func (sff *SmartFormFiller) PrintSupportedFields() {
	fmt.Println("=== 智能表单填充器支持的字段类型 ===")
	
	for patternKey, patterns := range sff.fieldPatterns {
		fmt.Printf("\n[%s] 匹配关键词:\n", patternKey)
		for i, pattern := range patterns {
			if i < 5 {
				fmt.Printf("  - %s\n", pattern)
			}
		}
		if len(patterns) > 5 {
			fmt.Printf("  ... 还有 %d 个关键词\n", len(patterns)-5)
		}
	}
	
	fmt.Println("\n=== Fuzz测试载荷类型 ===")
	for category, payloads := range sff.fuzzPayloads {
		fmt.Printf("[%s] %d 个载荷\n", category, len(payloads))
	}
}

