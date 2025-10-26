package core

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SmartParamValidator 智能参数验证器
// 解决问题：
// 1. 验证爆破的参数是否真实有效
// 2. 检测响应相似度，避免重复爆破
// 3. 及时停止无效的参数测试
type SmartParamValidator struct {
	mutex sync.RWMutex
	
	// HTTP客户端
	client *http.Client
	
	// 基准响应（用于对比）
	baselineResponses map[string]*ResponseSignature
	
	// 参数验证结果
	validParams map[string]*ParamValidation
	
	// 配置
	config ValidatorConfig
	
	// 统计
	stats ValidatorStats
}

// ValidatorConfig 验证器配置
type ValidatorConfig struct {
	// 响应相似度阈值（0-1），超过此值认为响应相同
	SimilarityThreshold float64
	
	// 连续相同响应的最大次数，超过后停止该参数的测试
	MaxSimilarResponses int
	
	// 最小响应差异（字节），小于此值认为响应相同
	MinResponseDiff int
	
	// 请求超时时间
	Timeout time.Duration
	
	// 是否启用验证
	Enabled bool
	
	// 最大并发请求数
	MaxConcurrency int
}

// ResponseSignature 响应签名
type ResponseSignature struct {
	StatusCode    int
	ContentLength int
	BodyHash      string // 响应体的MD5哈希
	Title         string // HTML标题
	ErrorPatterns []string // 错误特征
	HasContent    bool   // 是否有实质性内容
}

// ParamValidation 参数验证结果
type ParamValidation struct {
	ParamName        string
	IsValid          bool      // 是否有效（会影响响应）
	TestedValues     int       // 测试的值数量
	SimilarResponses int       // 相似响应数量
	ValidValues      []string  // 有效的参数值
	Signature        *ResponseSignature // 最后一次响应签名
	ShouldStop       bool      // 是否应该停止测试
	Reason           string    // 停止原因
}

// ValidatorStats 验证器统计
type ValidatorStats struct {
	TotalRequests     int
	ValidParams       int
	InvalidParams     int
	StoppedEarly      int // 提前停止的参数数
	SavedRequests     int // 节省的请求数
}

// NewSmartParamValidator 创建智能参数验证器
func NewSmartParamValidator(config ValidatorConfig) *SmartParamValidator {
	// 设置默认值
	if config.SimilarityThreshold == 0 {
		config.SimilarityThreshold = 0.95 // 95%相似度
	}
	if config.MaxSimilarResponses == 0 {
		config.MaxSimilarResponses = 3 // 连续3次相同响应就停止
	}
	if config.MinResponseDiff == 0 {
		config.MinResponseDiff = 50 // 最小50字节差异
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 5
	}
	
	return &SmartParamValidator{
		client: &http.Client{
			Timeout: config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// 允许重定向但记录
				return nil
			},
		},
		baselineResponses: make(map[string]*ResponseSignature),
		validParams:       make(map[string]*ParamValidation),
		config:            config,
		stats:             ValidatorStats{},
	}
}

// DefaultValidatorConfig 返回默认配置
func DefaultValidatorConfig() ValidatorConfig {
	return ValidatorConfig{
		SimilarityThreshold: 0.95,
		MaxSimilarResponses: 3,
		MinResponseDiff:     50,
		Timeout:             10 * time.Second,
		Enabled:             true,
		MaxConcurrency:      5,
	}
}

// ValidateFuzzedParams 验证爆破参数列表
// 返回：有效的URL列表
func (v *SmartParamValidator) ValidateFuzzedParams(baseURL string, fuzzedURLs []string) []string {
	if !v.config.Enabled || len(fuzzedURLs) == 0 {
		return fuzzedURLs
	}
	
	v.mutex.Lock()
	v.stats.TotalRequests = len(fuzzedURLs)
	v.mutex.Unlock()
	
	// 1. 获取基准响应（无参数或原始URL）
	baseSignature, err := v.getResponseSignature(baseURL)
	if err != nil {
		// 获取基准失败，返回所有URL（保守策略）
		return fuzzedURLs
	}
	
	v.mutex.Lock()
	v.baselineResponses[baseURL] = baseSignature
	v.mutex.Unlock()
	
	// 2. 按参数分组
	paramGroups := v.groupByParam(fuzzedURLs)
	
	// 3. 验证每个参数组
	validURLs := make([]string, 0)
	
	for paramName, urls := range paramGroups {
		fmt.Printf("  [参数验证] 测试参数: %s (%d 个值)\n", paramName, len(urls))
		
		validURLsForParam := v.validateParamGroup(paramName, urls, baseSignature)
		validURLs = append(validURLs, validURLsForParam...)
		
		v.mutex.RLock()
		validation := v.validParams[paramName]
		v.mutex.RUnlock()
		
		if validation != nil {
			if validation.IsValid {
				fmt.Printf("  [参数验证] ✓ %s 是有效参数，保留 %d 个URL\n", 
					paramName, len(validURLsForParam))
			} else {
				fmt.Printf("  [参数验证] ✗ %s 无效（%s），跳过 %d 个URL\n", 
					paramName, validation.Reason, len(urls))
				v.mutex.Lock()
				v.stats.InvalidParams++
				v.stats.SavedRequests += len(urls)
				v.mutex.Unlock()
			}
		}
	}
	
	v.mutex.Lock()
	savedRequests := len(fuzzedURLs) - len(validURLs)
	v.stats.SavedRequests = savedRequests
	v.mutex.Unlock()
	
	if savedRequests > 0 {
		fmt.Printf("  [参数验证] 节省 %d 个无效请求（原 %d → 现 %d）\n", 
			savedRequests, len(fuzzedURLs), len(validURLs))
	}
	
	return validURLs
}

// groupByParam 按参数名分组URL
func (v *SmartParamValidator) groupByParam(urls []string) map[string][]string {
	groups := make(map[string][]string)
	
	for _, urlStr := range urls {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			continue
		}
		
		params := parsedURL.Query()
		for paramName := range params {
			if groups[paramName] == nil {
				groups[paramName] = make([]string, 0)
			}
			groups[paramName] = append(groups[paramName], urlStr)
		}
	}
	
	return groups
}

// validateParamGroup 验证一组参数URL
func (v *SmartParamValidator) validateParamGroup(paramName string, urls []string, baseSignature *ResponseSignature) []string {
	validation := &ParamValidation{
		ParamName:    paramName,
		IsValid:      false,
		TestedValues: 0,
		ValidValues:  make([]string, 0),
	}
	
	validURLs := make([]string, 0)
	similarCount := 0
	
	for i, urlStr := range urls {
		// 获取响应签名
		signature, err := v.getResponseSignature(urlStr)
		if err != nil {
			continue
		}
		
		validation.TestedValues++
		validation.Signature = signature
		
		// 比较与基准响应的相似度
		similarity := v.calculateSimilarity(baseSignature, signature)
		
		if similarity < v.config.SimilarityThreshold {
			// 响应不同，说明参数有效
			validation.IsValid = true
			validation.ValidValues = append(validation.ValidValues, urlStr)
			validURLs = append(validURLs, urlStr)
			similarCount = 0 // 重置计数
		} else {
			// 响应相同
			similarCount++
			validation.SimilarResponses++
			
			// 连续相同响应超过阈值，停止测试
			if similarCount >= v.config.MaxSimilarResponses {
				validation.ShouldStop = true
				validation.Reason = fmt.Sprintf("连续%d次相同响应", similarCount)
				
				// 计算剩余未测试的数量
				remaining := len(urls) - i - 1
				if remaining > 0 {
					v.mutex.Lock()
					v.stats.StoppedEarly++
					v.stats.SavedRequests += remaining
					v.mutex.Unlock()
					
					fmt.Printf("  [参数验证] ⚠️  %s: %s，跳过剩余 %d 个值\n", 
						paramName, validation.Reason, remaining)
				}
				break
			}
		}
	}
	
	// 如果没有发现有效值，标记为无效参数
	if !validation.IsValid {
		if validation.TestedValues > 0 {
			validation.Reason = fmt.Sprintf("所有%d个值都返回相同响应", validation.TestedValues)
		} else {
			validation.Reason = "测试失败"
		}
	} else {
		v.mutex.Lock()
		v.stats.ValidParams++
		v.mutex.Unlock()
	}
	
	v.mutex.Lock()
	v.validParams[paramName] = validation
	v.mutex.Unlock()
	
	return validURLs
}

// getResponseSignature 获取URL的响应签名
func (v *SmartParamValidator) getResponseSignature(urlStr string) (*ResponseSignature, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	
	// 设置User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// 计算签名
	signature := &ResponseSignature{
		StatusCode:    resp.StatusCode,
		ContentLength: len(body),
		BodyHash:      v.calculateHash(body),
		HasContent:    len(body) > 0,
	}
	
	// 提取HTML标题
	bodyStr := string(body)
	if strings.Contains(bodyStr, "<title>") {
		start := strings.Index(bodyStr, "<title>") + 7
		end := strings.Index(bodyStr, "</title>")
		if end > start {
			signature.Title = bodyStr[start:end]
		}
	}
	
	// 检测错误特征
	errorPatterns := []string{
		"error", "exception", "warning", "not found", "404",
		"invalid", "forbidden", "unauthorized", "denied",
	}
	
	bodyLower := strings.ToLower(bodyStr)
	for _, pattern := range errorPatterns {
		if strings.Contains(bodyLower, pattern) {
			signature.ErrorPatterns = append(signature.ErrorPatterns, pattern)
		}
	}
	
	return signature, nil
}

// calculateSimilarity 计算两个响应的相似度（0-1）
func (v *SmartParamValidator) calculateSimilarity(sig1, sig2 *ResponseSignature) float64 {
	score := 0.0
	weights := 0.0
	
	// 1. 状态码相同 (权重: 0.3)
	if sig1.StatusCode == sig2.StatusCode {
		score += 0.3
	}
	weights += 0.3
	
	// 2. 内容长度差异 (权重: 0.3)
	lenDiff := abs(sig1.ContentLength - sig2.ContentLength)
	if lenDiff < v.config.MinResponseDiff {
		// 长度差异小于阈值
		score += 0.3
	} else {
		// 根据差异比例计算分数
		maxLen := maxInt(sig1.ContentLength, sig2.ContentLength)
		if maxLen > 0 {
			diffRatio := float64(lenDiff) / float64(maxLen)
			score += 0.3 * (1.0 - diffRatio)
		}
	}
	weights += 0.3
	
	// 3. 响应体哈希相同 (权重: 0.3)
	if sig1.BodyHash == sig2.BodyHash {
		score += 0.3
	}
	weights += 0.3
	
	// 4. 标题相同 (权重: 0.1)
	if sig1.Title != "" && sig1.Title == sig2.Title {
		score += 0.1
	}
	weights += 0.1
	
	return score / weights
}

// calculateHash 计算内容的MD5哈希
func (v *SmartParamValidator) calculateHash(data []byte) string {
	hasher := md5.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetStatistics 获取统计信息
func (v *SmartParamValidator) GetStatistics() ValidatorStats {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	return v.stats
}

// GetValidationResults 获取参数验证结果
func (v *SmartParamValidator) GetValidationResults() map[string]*ParamValidation {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	
	results := make(map[string]*ParamValidation)
	for k, v := range v.validParams {
		results[k] = v
	}
	return results
}

// PrintReport 打印验证报告
func (v *SmartParamValidator) PrintReport() {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("                    智能参数验证报告")
	fmt.Println(strings.Repeat("=", 70))
	
	fmt.Printf("\n【总体统计】\n")
	fmt.Printf("  处理参数数:     %d\n", len(v.validParams))
	fmt.Printf("  有效参数:       %d\n", v.stats.ValidParams)
	fmt.Printf("  无效参数:       %d\n", v.stats.InvalidParams)
	fmt.Printf("  提前停止:       %d\n", v.stats.StoppedEarly)
	fmt.Printf("  节省请求:       %d\n", v.stats.SavedRequests)
	
	if v.stats.TotalRequests > 0 {
		efficiency := float64(v.stats.SavedRequests) / float64(v.stats.TotalRequests) * 100
		fmt.Printf("  效率提升:       %.1f%%\n", efficiency)
	}
	
	fmt.Printf("\n【参数详情】\n")
	for _, validation := range v.validParams {
		status := "✗ 无效"
		if validation.IsValid {
			status = "✓ 有效"
		}
		
		fmt.Printf("  %s %s\n", status, validation.ParamName)
		fmt.Printf("    - 测试值数: %d\n", validation.TestedValues)
		fmt.Printf("    - 有效值数: %d\n", len(validation.ValidValues))
		
		if validation.ShouldStop {
			fmt.Printf("    - 提前停止: %s\n", validation.Reason)
		}
		
		if len(validation.ValidValues) > 0 && len(validation.ValidValues) <= 3 {
			fmt.Printf("    - 有效值: %v\n", validation.ValidValues)
		}
	}
	
	fmt.Println(strings.Repeat("=", 70))
}

// 辅助函数
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

