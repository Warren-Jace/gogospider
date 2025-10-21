package core

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
)

// PassiveCrawler 被动爬取器
type PassiveCrawler struct {
	mode          string // burp/har/proxy
	importedURLs  []string
	importedForms []*Form
	importedAPIs  []string
	statistics    *PassiveStats
}

// PassiveStats 被动爬取统计
type PassiveStats struct {
	ImportedRequests int
	ExtractedURLs    int
	ExtractedForms   int
	ExtractedAPIs    int
}

// BurpItem Burp Suite导出的XML格式
type BurpItem struct {
	XMLName  xml.Name `xml:"item"`
	Time     string   `xml:"time"`
	URL      string   `xml:"url"`
	Host     string   `xml:"host"`
	Port     string   `xml:"port"`
	Protocol string   `xml:"protocol"`
	Method   string   `xml:"method"`
	Path     string   `xml:"path"`
	Request  string   `xml:"request"`
	Response string   `xml:"response"`
	Status   string   `xml:"status"`
}

// BurpItems Burp XML根节点
type BurpItems struct {
	XMLName xml.Name    `xml:"items"`
	Items   []BurpItem  `xml:"item"`
}

// HAREntry HAR文件entry结构
type HAREntry struct {
	Request  HARRequest  `json:"request"`
	Response HARResponse `json:"response"`
}

// HARRequest HAR请求
type HARRequest struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	HTTPVersion string            `json:"httpVersion"`
	Headers     []HARHeader       `json:"headers"`
	QueryString []HARQueryString  `json:"queryString"`
	PostData    *HARPostData      `json:"postData"`
}

// HARResponse HAR响应
type HARResponse struct {
	Status  int         `json:"status"`
	Headers []HARHeader `json:"headers"`
	Content HARContent  `json:"content"`
}

// HARHeader HAR头
type HARHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARQueryString HAR查询字符串
type HARQueryString struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARPostData HAR POST数据
type HARPostData struct {
	MimeType string     `json:"mimeType"`
	Params   []HARParam `json:"params"`
	Text     string     `json:"text"`
}

// HARParam HAR参数
type HARParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARContent HAR内容
type HARContent struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// HAR HAR文件根结构
type HAR struct {
	Log HARLog `json:"log"`
}

// HARLog HAR日志
type HARLog struct {
	Version string     `json:"version"`
	Creator HARCreator `json:"creator"`
	Entries []HAREntry `json:"entries"`
}

// HARCreator HAR创建者
type HARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewPassiveCrawler 创建被动爬取器
func NewPassiveCrawler(mode string) *PassiveCrawler {
	return &PassiveCrawler{
		mode:          mode,
		importedURLs:  make([]string, 0),
		importedForms: make([]*Form, 0),
		importedAPIs:  make([]string, 0),
		statistics:    &PassiveStats{},
	}
}

// LoadFromBurp 从Burp Suite XML文件加载
func (pc *PassiveCrawler) LoadFromBurp(filename string) error {
	// 读取文件
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取Burp文件失败: %v", err)
	}
	
	// 解析XML
	var burpItems BurpItems
	err = xml.Unmarshal(data, &burpItems)
	if err != nil {
		return fmt.Errorf("解析Burp XML失败: %v", err)
	}
	
	// 处理每个请求
	for _, item := range burpItems.Items {
		pc.statistics.ImportedRequests++
		
		// 提取URL
		fullURL := item.URL
		if fullURL != "" {
			pc.importedURLs = append(pc.importedURLs, fullURL)
			pc.statistics.ExtractedURLs++
			
			// 判断是否为API
			if pc.isAPI(fullURL) {
				pc.importedAPIs = append(pc.importedAPIs, fullURL)
				pc.statistics.ExtractedAPIs++
			}
		}
		
		// 如果是POST请求，尝试提取表单
		if item.Method == "POST" {
			form := pc.extractFormFromBurpRequest(item)
			if form != nil {
				pc.importedForms = append(pc.importedForms, form)
				pc.statistics.ExtractedForms++
			}
		}
	}
	
	fmt.Printf("从Burp Suite导入: %d个请求, %d个URL, %d个表单, %d个API\n",
		pc.statistics.ImportedRequests,
		pc.statistics.ExtractedURLs,
		pc.statistics.ExtractedForms,
		pc.statistics.ExtractedAPIs)
	
	return nil
}

// LoadFromHAR 从HAR文件加载
func (pc *PassiveCrawler) LoadFromHAR(filename string) error {
	// 读取文件
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取HAR文件失败: %v", err)
	}
	
	// 解析JSON
	var har HAR
	err = json.Unmarshal(data, &har)
	if err != nil {
		return fmt.Errorf("解析HAR JSON失败: %v", err)
	}
	
	// 处理每个entry
	for _, entry := range har.Log.Entries {
		pc.statistics.ImportedRequests++
		
		// 提取URL
		if entry.Request.URL != "" {
			pc.importedURLs = append(pc.importedURLs, entry.Request.URL)
			pc.statistics.ExtractedURLs++
			
			// 判断是否为API
			if pc.isAPI(entry.Request.URL) {
				pc.importedAPIs = append(pc.importedAPIs, entry.Request.URL)
				pc.statistics.ExtractedAPIs++
			}
		}
		
		// 如果是POST请求，提取表单
		if entry.Request.Method == "POST" && entry.Request.PostData != nil {
			form := pc.extractFormFromHARRequest(entry.Request)
			if form != nil {
				pc.importedForms = append(pc.importedForms, form)
				pc.statistics.ExtractedForms++
			}
		}
	}
	
	fmt.Printf("从HAR文件导入: %d个请求, %d个URL, %d个表单, %d个API\n",
		pc.statistics.ImportedRequests,
		pc.statistics.ExtractedURLs,
		pc.statistics.ExtractedForms,
		pc.statistics.ExtractedAPIs)
	
	return nil
}

// extractFormFromBurpRequest 从Burp请求中提取表单
func (pc *PassiveCrawler) extractFormFromBurpRequest(item BurpItem) *Form {
	// 简化实现：从URL和请求内容中提取
	form := &Form{
		Action: item.URL,
		Method: item.Method,
		Fields: make([]FormField, 0),
	}
	
	// 解析请求体（简化处理）
	if strings.Contains(item.Request, "Content-Type: application/x-www-form-urlencoded") {
		// 查找请求体
		parts := strings.Split(item.Request, "\r\n\r\n")
		if len(parts) > 1 {
			body := parts[1]
			params := strings.Split(body, "&")
			
			for _, param := range params {
				parts := strings.SplitN(param, "=", 2)
				if len(parts) == 2 {
					field := FormField{
						Name:  parts[0],
						Type:  "text",
						Value: parts[1],
					}
					form.Fields = append(form.Fields, field)
				}
			}
		}
	}
	
	if len(form.Fields) > 0 {
		return form
	}
	
	return nil
}

// extractFormFromHARRequest 从HAR请求中提取表单
func (pc *PassiveCrawler) extractFormFromHARRequest(request HARRequest) *Form {
	form := &Form{
		Action: request.URL,
		Method: request.Method,
		Fields: make([]FormField, 0),
	}
	
	// 从PostData提取参数
	if request.PostData != nil {
		for _, param := range request.PostData.Params {
			field := FormField{
				Name:  param.Name,
				Type:  "text",
				Value: param.Value,
			}
			form.Fields = append(form.Fields, field)
		}
	}
	
	if len(form.Fields) > 0 {
		return form
	}
	
	return nil
}

// isAPI 判断是否为API端点
func (pc *PassiveCrawler) isAPI(urlStr string) bool {
	urlLower := strings.ToLower(urlStr)
	
	apiPatterns := []string{
		"/api/", "/v1/", "/v2/", "/v3/",
		"/rest/", "/graphql", "/json",
		"/ajax/", "/xhr/",
	}
	
	for _, pattern := range apiPatterns {
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}
	
	// 检查Content-Type（如果在响应头中）
	// 这里简化处理
	
	return false
}

// GetImportedURLs 获取导入的URL
func (pc *PassiveCrawler) GetImportedURLs() []string {
	return pc.importedURLs
}

// GetImportedForms 获取导入的表单
func (pc *PassiveCrawler) GetImportedForms() []*Form {
	return pc.importedForms
}

// GetImportedAPIs 获取导入的API
func (pc *PassiveCrawler) GetImportedAPIs() []string {
	return pc.importedAPIs
}

// GetStatistics 获取统计信息
func (pc *PassiveCrawler) GetStatistics() *PassiveStats {
	return pc.statistics
}

// FilterByDomain 按域名过滤URL
func (pc *PassiveCrawler) FilterByDomain(targetDomain string) []string {
	filtered := make([]string, 0)
	
	for _, urlStr := range pc.importedURLs {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			continue
		}
		
		// 检查是否为目标域名或子域名
		if parsedURL.Host == targetDomain || 
		   strings.HasSuffix(parsedURL.Host, "."+targetDomain) {
			filtered = append(filtered, urlStr)
		}
	}
	
	return filtered
}

// ExportToResult 导出为爬取结果格式
func (pc *PassiveCrawler) ExportToResult(targetURL string) *Result {
	result := &Result{
		URL:         targetURL,
		StatusCode:  200,
		ContentType: "passive/import",
		Links:       pc.importedURLs,
		Forms:       make([]Form, 0),
		APIs:        pc.importedAPIs,
		Assets:      make([]string, 0),
	}
	
	// 转换表单格式
	for _, form := range pc.importedForms {
		result.Forms = append(result.Forms, *form)
	}
	
	return result
}

// GenerateImportReport 生成导入报告
func (pc *PassiveCrawler) GenerateImportReport() string {
	var report strings.Builder
	
	report.WriteString("=== 被动爬取导入报告 ===\n\n")
	report.WriteString(fmt.Sprintf("导入模式: %s\n", pc.mode))
	report.WriteString(fmt.Sprintf("导入请求数: %d\n", pc.statistics.ImportedRequests))
	report.WriteString(fmt.Sprintf("提取URL数: %d\n", pc.statistics.ExtractedURLs))
	report.WriteString(fmt.Sprintf("提取表单数: %d\n", pc.statistics.ExtractedForms))
	report.WriteString(fmt.Sprintf("提取API数: %d\n", pc.statistics.ExtractedAPIs))
	report.WriteString("\n")
	
	// 显示部分URL示例
	if len(pc.importedURLs) > 0 {
		report.WriteString("URL示例:\n")
		maxShow := 10
		if len(pc.importedURLs) < maxShow {
			maxShow = len(pc.importedURLs)
		}
		
		for i := 0; i < maxShow; i++ {
			report.WriteString(fmt.Sprintf("  %d. %s\n", i+1, pc.importedURLs[i]))
		}
		
		if len(pc.importedURLs) > maxShow {
			report.WriteString(fmt.Sprintf("  ... 还有 %d 个URL\n", len(pc.importedURLs)-maxShow))
		}
	}
	
	return report.String()
}

