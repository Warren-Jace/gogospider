package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// PipelineMode 管道模式（支持stdin/stdout）
type PipelineMode struct {
	inputReader  io.Reader
	outputWriter io.Writer
	errorWriter  io.Writer
	
	// 配置
	inputFormat  string // "text" 或 "json"
	outputFormat string // "text", "json", "jsonl"
	
	// 统计
	processedCount int
	errorCount     int
	mutex          sync.Mutex
}

// PipelineModeConfig 管道模式配置
type PipelineModeConfig struct {
	EnableStdin      bool   // 启用标准输入
	EnableStdout     bool   // 启用标准输出
	InputFormat      string // 输入格式
	OutputFormat     string // 输出格式
	Quiet            bool   // 静默模式（不输出日志到stderr）
}

// NewPipelineMode 创建管道模式
func NewPipelineMode(config PipelineModeConfig) *PipelineMode {
	pm := &PipelineMode{
		inputReader:  os.Stdin,
		outputWriter: os.Stdout,
		errorWriter:  os.Stderr,
		inputFormat:  config.InputFormat,
		outputFormat: config.OutputFormat,
	}
	
	// 设置默认格式
	if pm.inputFormat == "" {
		pm.inputFormat = "text"
	}
	if pm.outputFormat == "" {
		pm.outputFormat = "text"
	}
	
	// 静默模式：丢弃错误输出
	if config.Quiet {
		pm.errorWriter = io.Discard
	}
	
	return pm
}

// ReadURLs 从stdin读取URL
func (pm *PipelineMode) ReadURLs() ([]string, error) {
	urls := make([]string, 0)
	scanner := bufio.NewScanner(pm.inputReader)
	
	// 设置更大的缓冲区（处理长URL）
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// 根据输入格式解析
		switch pm.inputFormat {
		case "text":
			urls = append(urls, line)
			
		case "json":
			// 解析JSON格式的输入
			var input struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal([]byte(line), &input); err != nil {
				fmt.Fprintf(pm.errorWriter, "解析JSON失败: %v\n", err)
				continue
			}
			if input.URL != "" {
				urls = append(urls, input.URL)
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	return urls, nil
}

// WriteURL 写入单个URL到stdout
func (pm *PipelineMode) WriteURL(url string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	var output string
	
	switch pm.outputFormat {
	case "text":
		output = url + "\n"
		
	case "json":
		data := map[string]string{"url": url}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		output = string(jsonData) + "\n"
		
	case "jsonl":
		// JSONL格式（每行一个JSON对象）
		data := map[string]string{"url": url}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		output = string(jsonData) + "\n"
	}
	
	_, err := pm.outputWriter.Write([]byte(output))
	if err != nil {
		return err
	}
	
	pm.processedCount++
	return nil
}

// WriteURLs 批量写入URL
func (pm *PipelineMode) WriteURLs(urls []string) error {
	for _, url := range urls {
		if err := pm.WriteURL(url); err != nil {
			pm.errorCount++
			fmt.Fprintf(pm.errorWriter, "写入URL失败: %v\n", err)
		}
	}
	return nil
}

// WriteJSON 写入JSON格式的结果
func (pm *PipelineMode) WriteJSON(data interface{}) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(pm.outputWriter, "%s\n", jsonData)
	if err != nil {
		return err
	}
	
	pm.processedCount++
	return nil
}

// Log 写入日志到stderr（不影响stdout）
func (pm *PipelineMode) Log(format string, args ...interface{}) {
	fmt.Fprintf(pm.errorWriter, format+"\n", args...)
}

// GetStatistics 获取统计信息
func (pm *PipelineMode) GetStatistics() PipelineStats {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	return PipelineStats{
		ProcessedCount: pm.processedCount,
		ErrorCount:     pm.errorCount,
	}
}

// PipelineStats 管道统计信息
type PipelineStats struct {
	ProcessedCount int
	ErrorCount     int
}

// PipelineProcessor 管道处理器（用于工具链集成）
type PipelineProcessor struct {
	spider       *Spider
	pipelineMode *PipelineMode
	config       PipelineModeConfig
}

// NewPipelineProcessor 创建管道处理器
func NewPipelineProcessor(spider *Spider, config PipelineModeConfig) *PipelineProcessor {
	return &PipelineProcessor{
		spider:       spider,
		pipelineMode: NewPipelineMode(config),
		config:       config,
	}
}

// Process 处理管道输入
func (pp *PipelineProcessor) Process() error {
	// 1. 从stdin读取URL
	urls, err := pp.pipelineMode.ReadURLs()
	if err != nil {
		return fmt.Errorf("读取输入失败: %v", err)
	}
	
	if len(urls) == 0 {
		pp.pipelineMode.Log("⚠️  未收到任何URL输入")
		return nil
	}
	
	pp.pipelineMode.Log("📥 收到 %d 个URL", len(urls))
	
	// 2. 处理每个URL
	for i, url := range urls {
		pp.pipelineMode.Log("🔍 正在处理 [%d/%d]: %s", i+1, len(urls), url)
		
		// 爬取URL
		results, err := pp.crawlURL(url)
		if err != nil {
			pp.pipelineMode.Log("❌ 爬取失败: %v", err)
			continue
		}
		
		// 输出结果
		if err := pp.outputResults(results); err != nil {
			pp.pipelineMode.Log("❌ 输出失败: %v", err)
			continue
		}
		
		pp.pipelineMode.Log("✅ 完成 [%d/%d]", i+1, len(urls))
	}
	
	// 3. 打印统计
	stats := pp.pipelineMode.GetStatistics()
	pp.pipelineMode.Log("\n📊 统计: 处理=%d, 错误=%d", stats.ProcessedCount, stats.ErrorCount)
	
	return nil
}

// crawlURL 爬取单个URL（完整实现）
func (pp *PipelineProcessor) crawlURL(targetURL string) ([]*Result, error) {
	if pp.spider == nil {
		return nil, fmt.Errorf("spider未初始化")
	}
	
	// 启动爬取
	if err := pp.spider.Start(targetURL); err != nil {
		return nil, fmt.Errorf("爬取失败: %v", err)
	}
	
	// 获取结果
	results := pp.spider.GetResults()
	return results, nil
}

// outputResults 输出结果
func (pp *PipelineProcessor) outputResults(results []*Result) error {
	for _, result := range results {
		// 输出所有发现的链接
		for _, link := range result.Links {
			if err := pp.pipelineMode.WriteURL(link); err != nil {
				return err
			}
		}
		
		// 输出所有发现的API
		for _, api := range result.APIs {
			if err := pp.pipelineMode.WriteURL(api); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// PipelineChain 管道链（支持多个工具串联）
type PipelineChain struct {
	processors []PipelineProcessor
}

// NewPipelineChain 创建管道链
func NewPipelineChain() *PipelineChain {
	return &PipelineChain{
		processors: make([]PipelineProcessor, 0),
	}
}

// Add 添加处理器
func (pc *PipelineChain) Add(processor PipelineProcessor) {
	pc.processors = append(pc.processors, processor)
}

// Execute 执行管道链
func (pc *PipelineChain) Execute() error {
	for _, processor := range pc.processors {
		if err := processor.Process(); err != nil {
			return err
		}
	}
	return nil
}

// StdinHelper stdin辅助函数
type StdinHelper struct{}

// IsStdinAvailable 检查stdin是否有数据
func (sh *StdinHelper) IsStdinAvailable() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	
	// 检查是否是管道或重定向
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// ReadLines 读取所有行
func (sh *StdinHelper) ReadLines() ([]string, error) {
	lines := make([]string, 0)
	scanner := bufio.NewScanner(os.Stdin)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	return lines, nil
}

// StdoutHelper stdout辅助函数
type StdoutHelper struct{}

// WriteLine 写入一行
func (sh *StdoutHelper) WriteLine(line string) error {
	_, err := fmt.Println(line)
	return err
}

// WriteJSON 写入JSON
func (sh *StdoutHelper) WriteJSON(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	_, err = fmt.Println(string(jsonData))
	return err
}

