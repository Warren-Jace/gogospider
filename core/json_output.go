package core

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// JSONOutput JSON输出结构（参考Katana设计）
type JSONOutput struct {
	// 基础信息
	URL        string `json:"url"`
	Method     string `json:"method"`
	StatusCode int    `json:"status_code,omitempty"`
	
	// 来源信息
	Source     string `json:"source"`      // href, script, form, ajax等
	SourceURL  string `json:"source_url"`  // 从哪个URL发现的
	Tag        string `json:"tag"`         // HTML标签类型
	Attribute  string `json:"attribute"`   // 属性名称
	
	// 层级信息
	Depth      int    `json:"depth"`
	
	// 时间信息
	Timestamp  string `json:"timestamp"`
	
	// 扩展信息
	ContentType string            `json:"content_type,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	
	// 发现的资源
	Links       []string `json:"links,omitempty"`
	Forms       []string `json:"forms,omitempty"`
	APIs        []string `json:"apis,omitempty"`
	
	// 安全相关
	Sensitive   []string `json:"sensitive,omitempty"`
	Technologies []string `json:"technologies,omitempty"`
}

// JSONExporter JSON导出器
type JSONExporter struct {
	outputFile string
	mode       string // "line" 或 "array"
	file       *os.File
	encoder    *json.Encoder
}

// NewJSONExporter 创建JSON导出器
func NewJSONExporter(outputFile string, mode string) (*JSONExporter, error) {
	if mode != "line" && mode != "array" {
		mode = "line" // 默认行分隔JSON (NDJSON)
	}
	
	file, err := os.Create(outputFile)
	if err != nil {
		return nil, fmt.Errorf("创建输出文件失败: %v", err)
	}
	
	exporter := &JSONExporter{
		outputFile: outputFile,
		mode:       mode,
		file:       file,
		encoder:    json.NewEncoder(file),
	}
	
	// 如果是数组模式，写入开始符号
	if mode == "array" {
		file.WriteString("[\n")
	}
	
	return exporter, nil
}

// Export 导出单个结果
func (je *JSONExporter) Export(output *JSONOutput) error {
	return je.encoder.Encode(output)
}

// ExportBatch 批量导出
func (je *JSONExporter) ExportBatch(outputs []*JSONOutput) error {
	for i, output := range outputs {
		if je.mode == "array" && i > 0 {
			je.file.WriteString(",\n")
		}
		if err := je.Export(output); err != nil {
			return err
		}
	}
	return nil
}

// Close 关闭导出器
func (je *JSONExporter) Close() error {
	if je.mode == "array" {
		je.file.WriteString("]\n")
	}
	return je.file.Close()
}

// ConvertResultToJSON 将Result转换为JSON输出格式
func ConvertResultToJSON(result *Result, depth int, sourceURL string) *JSONOutput {
	output := &JSONOutput{
		URL:         result.URL,
		Method:      "GET",
		StatusCode:  result.StatusCode,
		Source:      "crawler",
		SourceURL:   sourceURL,
		Tag:         "",
		Attribute:   "",
		Depth:       depth,
		Timestamp:   time.Now().Format(time.RFC3339),
		ContentType: result.ContentType,
		Headers:     result.Headers,
		Links:       result.Links,
		APIs:        result.APIs,
	}
	
	// 转换表单为字符串数组
	if len(result.Forms) > 0 {
		output.Forms = make([]string, len(result.Forms))
		for i, form := range result.Forms {
			output.Forms[i] = form.Action
		}
	}
	
	return output
}

// JSONOutputFormatter JSON输出格式化器
type JSONOutputFormatter struct {
	mode       string // "compact", "pretty", "line"
	includeAll bool   // 是否包含所有字段
}

// NewJSONOutputFormatter 创建格式化器
func NewJSONOutputFormatter(mode string, includeAll bool) *JSONOutputFormatter {
	return &JSONOutputFormatter{
		mode:       mode,
		includeAll: includeAll,
	}
}

// Format 格式化输出
func (jof *JSONOutputFormatter) Format(output *JSONOutput) (string, error) {
	var data []byte
	var err error
	
	switch jof.mode {
	case "pretty":
		data, err = json.MarshalIndent(output, "", "  ")
	case "compact":
		data, err = json.Marshal(output)
	case "line":
		data, err = json.Marshal(output)
	default:
		data, err = json.Marshal(output)
	}
	
	if err != nil {
		return "", err
	}
	
	return string(data), nil
}

// JSONStreamWriter JSON流式写入器（支持大规模输出）
type JSONStreamWriter struct {
	file    *os.File
	encoder *json.Encoder
	count   int
}

// NewJSONStreamWriter 创建流式写入器
func NewJSONStreamWriter(filename string) (*JSONStreamWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	
	return &JSONStreamWriter{
		file:    file,
		encoder: json.NewEncoder(file),
		count:   0,
	}, nil
}

// Write 写入一条记录
func (jsw *JSONStreamWriter) Write(output *JSONOutput) error {
	jsw.count++
	return jsw.encoder.Encode(output)
}

// GetCount 获取写入数量
func (jsw *JSONStreamWriter) GetCount() int {
	return jsw.count
}

// Close 关闭写入器
func (jsw *JSONStreamWriter) Close() error {
	return jsw.file.Close()
}

// JSONLinesWriter JSONL格式写入器（每行一个JSON对象）
type JSONLinesWriter struct {
	file    *os.File
	count   int
	written map[string]bool // 去重
}

// NewJSONLinesWriter 创建JSONL写入器
func NewJSONLinesWriter(filename string) (*JSONLinesWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	
	return &JSONLinesWriter{
		file:    file,
		count:   0,
		written: make(map[string]bool),
	}, nil
}

// WriteLine 写入一行（带去重）
func (jlw *JSONLinesWriter) WriteLine(output *JSONOutput) error {
	// 简单去重：基于URL
	if jlw.written[output.URL] {
		return nil
	}
	jlw.written[output.URL] = true
	
	data, err := json.Marshal(output)
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(jlw.file, "%s\n", data)
	if err != nil {
		return err
	}
	
	jlw.count++
	return nil
}

// GetCount 获取写入数量
func (jlw *JSONLinesWriter) GetCount() int {
	return jlw.count
}

// Close 关闭写入器
func (jlw *JSONLinesWriter) Close() error {
	return jlw.file.Close()
}

