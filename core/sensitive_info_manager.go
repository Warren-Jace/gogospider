package core

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// SensitiveInfoManager 统一的敏感信息管理器
// 提供集中化的敏感信息收集、分类、去重、导出功能
type SensitiveInfoManager struct {
	detector       *SensitiveInfoDetector
	targetDomain   string
	scanStartTime  time.Time
	outputDir      string // 统一输出目录
	baseFilename   string // 基础文件名（不含扩展名）
	mutex          sync.Mutex
	
	// 统计信息
	totalScanned   int
	uniqueFindings map[string]*SensitiveInfo // 去重后的发现（key: type+value+url）
}

// SensitiveInfoManagerConfig 管理器配置
type SensitiveInfoManagerConfig struct {
	TargetDomain  string
	OutputDir     string
	BaseFilename  string
	Detector      *SensitiveInfoDetector
}

// NewSensitiveInfoManager 创建敏感信息管理器
func NewSensitiveInfoManager(cfg SensitiveInfoManagerConfig) *SensitiveInfoManager {
	if cfg.OutputDir == "" {
		cfg.OutputDir = "."
	}
	
	if cfg.BaseFilename == "" {
		timestamp := time.Now().Format("20060102_150405")
		domain := strings.ReplaceAll(cfg.TargetDomain, ":", "_")
		domain = strings.ReplaceAll(domain, "/", "_")
		cfg.BaseFilename = fmt.Sprintf("sensitive_%s_%s", domain, timestamp)
	}
	
	return &SensitiveInfoManager{
		detector:       cfg.Detector,
		targetDomain:   cfg.TargetDomain,
		scanStartTime:  time.Now(),
		outputDir:      cfg.OutputDir,
		baseFilename:   cfg.BaseFilename,
		uniqueFindings: make(map[string]*SensitiveInfo),
	}
}

// CollectFindings 收集并去重敏感信息
func (sim *SensitiveInfoManager) CollectFindings() {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	
	if sim.detector == nil {
		return
	}
	
	findings := sim.detector.GetFindings()
	
	// 去重逻辑：相同类型 + 相同值 + 相同URL = 重复
	for _, finding := range findings {
		key := fmt.Sprintf("%s|%s|%s", finding.Type, finding.FullValue, finding.SourceURL)
		if _, exists := sim.uniqueFindings[key]; !exists {
			sim.uniqueFindings[key] = finding
		}
	}
}

// GetStatistics 获取统计信息
func (sim *SensitiveInfoManager) GetStatistics() map[string]interface{} {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	
	stats := make(map[string]interface{})
	
	// 基本统计
	stats["total_unique_findings"] = len(sim.uniqueFindings)
	stats["scan_start_time"] = sim.scanStartTime.Format("2006-01-02 15:04:05")
	stats["target_domain"] = sim.targetDomain
	
	// 按严重程度统计
	severityCount := make(map[string]int)
	typeCount := make(map[string]int)
	urlCount := make(map[string]int)
	
	for _, finding := range sim.uniqueFindings {
		severityCount[finding.Severity]++
		typeCount[finding.Type]++
		urlCount[finding.SourceURL]++
	}
	
	stats["by_severity"] = severityCount
	stats["by_type"] = typeCount
	stats["affected_urls_count"] = len(urlCount)
	stats["most_affected_urls"] = sim.getMostAffectedURLs(urlCount, 10)
	
	return stats
}

// getMostAffectedURLs 获取受影响最多的URL
func (sim *SensitiveInfoManager) getMostAffectedURLs(urlCount map[string]int, limit int) []map[string]interface{} {
	type urlStat struct {
		URL   string
		Count int
	}
	
	var stats []urlStat
	for url, count := range urlCount {
		stats = append(stats, urlStat{URL: url, Count: count})
	}
	
	// 按数量降序排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})
	
	// 限制数量
	if len(stats) > limit {
		stats = stats[:limit]
	}
	
	result := make([]map[string]interface{}, len(stats))
	for i, stat := range stats {
		result[i] = map[string]interface{}{
			"url":   stat.URL,
			"count": stat.Count,
		}
	}
	
	return result
}

// ExportAll 导出所有格式的报告
func (sim *SensitiveInfoManager) ExportAll() error {
	if len(sim.uniqueFindings) == 0 {
		fmt.Println("[敏感信息] 未发现敏感信息，跳过导出")
		return nil
	}
	
	// 确保输出目录存在
	if err := os.MkdirAll(sim.outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}
	
	errors := make([]error, 0)
	
	// 1. 导出文本报告
	if err := sim.ExportText(); err != nil {
		errors = append(errors, fmt.Errorf("导出文本报告失败: %v", err))
	}
	
	// 2. 导出JSON报告
	if err := sim.ExportJSON(); err != nil {
		errors = append(errors, fmt.Errorf("导出JSON报告失败: %v", err))
	}
	
	// 3. 导出CSV报告
	if err := sim.ExportCSV(); err != nil {
		errors = append(errors, fmt.Errorf("导出CSV报告失败: %v", err))
	}
	
	// 4. 导出HTML报告
	if err := sim.ExportHTML(); err != nil {
		errors = append(errors, fmt.Errorf("导出HTML报告失败: %v", err))
	}
	
	// 5. 导出摘要报告
	if err := sim.ExportSummary(); err != nil {
		errors = append(errors, fmt.Errorf("导出摘要报告失败: %v", err))
	}
	
	if len(errors) > 0 {
		var errorMsg string
		for _, err := range errors {
			errorMsg += err.Error() + "\n"
		}
		return fmt.Errorf("部分导出失败:\n%s", errorMsg)
	}
	
	sim.printExportSummary()
	return nil
}

// ExportText 导出文本格式报告
func (sim *SensitiveInfoManager) ExportText() error {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	
	filename := filepath.Join(sim.outputDir, sim.baseFilename+".txt")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// 写入文件头
	file.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	file.WriteString("║             敏感信息泄露检测报告 (文本格式)                    ║\n")
	file.WriteString("╚══════════════════════════════════════════════════════════════╝\n\n")
	
	// 写入统计信息
	stats := sim.GetStatistics()
	file.WriteString(fmt.Sprintf("【扫描概况】\n"))
	file.WriteString(fmt.Sprintf("  目标域名: %s\n", stats["target_domain"]))
	file.WriteString(fmt.Sprintf("  扫描时间: %s\n", stats["scan_start_time"]))
	file.WriteString(fmt.Sprintf("  发现总数: %d（已去重）\n", stats["total_unique_findings"]))
	file.WriteString(fmt.Sprintf("  受影响URL数: %d\n\n", stats["affected_urls_count"]))
	
	// 按严重程度统计
	severityStats := stats["by_severity"].(map[string]int)
	file.WriteString("【严重程度分布】\n")
	file.WriteString(fmt.Sprintf("  🔴 高危 (HIGH):   %d\n", severityStats["HIGH"]))
	file.WriteString(fmt.Sprintf("  🟡 中危 (MEDIUM): %d\n", severityStats["MEDIUM"]))
	file.WriteString(fmt.Sprintf("  🟢 低危 (LOW):    %d\n\n", severityStats["LOW"]))
	
	// 按类型统计
	typeStats := stats["by_type"].(map[string]int)
	file.WriteString("【类型分布】\n")
	for infoType, count := range typeStats {
		file.WriteString(fmt.Sprintf("  - %-30s: %d\n", infoType, count))
	}
	file.WriteString("\n")
	
	file.WriteString(strings.Repeat("═", 64) + "\n\n")
	
	// 按严重程度分组输出详细信息
	sim.writeDetailsByServerity(file, "HIGH", "🔴 高危发现")
	sim.writeDetailsByServerity(file, "MEDIUM", "🟡 中危发现")
	sim.writeDetailsByServerity(file, "LOW", "🟢 低危发现")
	
	return nil
}

// writeDetailsByServerity 按严重程度写入详细信息
func (sim *SensitiveInfoManager) writeDetailsByServerity(file *os.File, severity string, title string) {
	findings := sim.getFindingsBySeverity(severity)
	if len(findings) == 0 {
		return
	}
	
	file.WriteString(fmt.Sprintf("%s（共 %d 项）\n", title, len(findings)))
	file.WriteString(strings.Repeat("-", 64) + "\n\n")
	
	for i, finding := range findings {
		file.WriteString(fmt.Sprintf("[%d] %s\n", i+1, finding.Type))
		file.WriteString(fmt.Sprintf("    来源URL: %s\n", finding.SourceURL))
		file.WriteString(fmt.Sprintf("    位置: %s\n", finding.Location))
		file.WriteString(fmt.Sprintf("    值: %s\n", finding.Value))
		if finding.FullValue != finding.Value {
			file.WriteString(fmt.Sprintf("    完整值: %s\n", finding.FullValue))
		}
		file.WriteString("\n")
	}
	
	file.WriteString("\n")
}

// ExportJSON 导出JSON格式报告
func (sim *SensitiveInfoManager) ExportJSON() error {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	
	filename := filepath.Join(sim.outputDir, sim.baseFilename+".json")
	
	stats := sim.GetStatistics()
	findings := make([]*SensitiveInfo, 0, len(sim.uniqueFindings))
	for _, finding := range sim.uniqueFindings {
		findings = append(findings, finding)
	}
	
	// 按严重程度和类型排序
	sort.Slice(findings, func(i, j int) bool {
		// 优先级：HIGH > MEDIUM > LOW
		severityOrder := map[string]int{"HIGH": 0, "MEDIUM": 1, "LOW": 2}
		if severityOrder[findings[i].Severity] != severityOrder[findings[j].Severity] {
			return severityOrder[findings[i].Severity] < severityOrder[findings[j].Severity]
		}
		return findings[i].Type < findings[j].Type
	})
	
	report := map[string]interface{}{
		"report_version": "1.0",
		"report_type":    "Sensitive Information Disclosure Report",
		"generated_at":   time.Now().Format(time.RFC3339),
		"statistics":     stats,
		"findings":       findings,
	}
	
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON编码失败: %v", err)
	}
	
	return os.WriteFile(filename, data, 0644)
}

// ExportCSV 导出CSV格式报告
func (sim *SensitiveInfoManager) ExportCSV() error {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	
	filename := filepath.Join(sim.outputDir, sim.baseFilename+".csv")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// 写入UTF-8 BOM（解决Excel中文乱码）
	file.Write([]byte{0xEF, 0xBB, 0xBF})
	
	// 写入表头
	headers := []string{"序号", "严重程度", "类型", "来源URL", "位置", "脱敏值", "完整值"}
	if err := writer.Write(headers); err != nil {
		return err
	}
	
	// 获取并排序发现
	findings := make([]*SensitiveInfo, 0, len(sim.uniqueFindings))
	for _, finding := range sim.uniqueFindings {
		findings = append(findings, finding)
	}
	
	sort.Slice(findings, func(i, j int) bool {
		severityOrder := map[string]int{"HIGH": 0, "MEDIUM": 1, "LOW": 2}
		if severityOrder[findings[i].Severity] != severityOrder[findings[j].Severity] {
			return severityOrder[findings[i].Severity] < severityOrder[findings[j].Severity]
		}
		return findings[i].Type < findings[j].Type
	})
	
	// 写入数据
	for i, finding := range findings {
		record := []string{
			fmt.Sprintf("%d", i+1),
			finding.Severity,
			finding.Type,
			finding.SourceURL,
			finding.Location,
			finding.Value,
			finding.FullValue,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	
	return nil
}

// ExportHTML 导出HTML格式报告
func (sim *SensitiveInfoManager) ExportHTML() error {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	
	filename := filepath.Join(sim.outputDir, sim.baseFilename+".html")
	
	tmpl := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>敏感信息泄露检测报告</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 10px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 { font-size: 2em; margin-bottom: 10px; }
        .header .subtitle { opacity: 0.9; }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            padding: 30px;
            background: #f8f9fa;
        }
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .stat-card .number {
            font-size: 2.5em;
            font-weight: bold;
            margin: 10px 0;
        }
        .stat-card .label { color: #666; }
        .severity-high .number { color: #e74c3c; }
        .severity-medium .number { color: #f39c12; }
        .severity-low .number { color: #27ae60; }
        .findings {
            padding: 30px;
        }
        .severity-section {
            margin-bottom: 40px;
        }
        .severity-section h2 {
            padding: 15px;
            border-radius: 8px;
            margin-bottom: 20px;
        }
        .severity-high h2 {
            background: #e74c3c;
            color: white;
        }
        .severity-medium h2 {
            background: #f39c12;
            color: white;
        }
        .severity-low h2 {
            background: #27ae60;
            color: white;
        }
        .finding-card {
            background: #f8f9fa;
            border-left: 4px solid #667eea;
            padding: 20px;
            margin-bottom: 15px;
            border-radius: 5px;
        }
        .finding-card h3 {
            color: #2c3e50;
            margin-bottom: 10px;
        }
        .finding-detail {
            margin: 8px 0;
            color: #555;
        }
        .finding-detail strong {
            display: inline-block;
            width: 100px;
            color: #2c3e50;
        }
        .finding-value {
            background: #fff;
            padding: 8px;
            border-radius: 4px;
            font-family: 'Courier New', monospace;
            word-break: break-all;
        }
        .footer {
            background: #2c3e50;
            color: white;
            text-align: center;
            padding: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔒 敏感信息泄露检测报告</h1>
            <div class="subtitle">目标域名: {{.TargetDomain}}</div>
            <div class="subtitle">生成时间: {{.GeneratedTime}}</div>
        </div>
        
        <div class="stats">
            <div class="stat-card">
                <div class="label">发现总数</div>
                <div class="number">{{.TotalFindings}}</div>
            </div>
            <div class="stat-card severity-high">
                <div class="label">高危</div>
                <div class="number">{{.HighCount}}</div>
            </div>
            <div class="stat-card severity-medium">
                <div class="label">中危</div>
                <div class="number">{{.MediumCount}}</div>
            </div>
            <div class="stat-card severity-low">
                <div class="label">低危</div>
                <div class="number">{{.LowCount}}</div>
            </div>
            <div class="stat-card">
                <div class="label">受影响URL</div>
                <div class="number">{{.AffectedURLs}}</div>
            </div>
        </div>
        
        <div class="findings">
            {{if .HighFindings}}
            <div class="severity-section severity-high">
                <h2>🔴 高危发现 ({{len .HighFindings}})</h2>
                {{range $index, $finding := .HighFindings}}
                <div class="finding-card">
                    <h3>{{add $index 1}}. {{$finding.Type}}</h3>
                    <div class="finding-detail"><strong>来源URL:</strong> {{$finding.SourceURL}}</div>
                    <div class="finding-detail"><strong>位置:</strong> {{$finding.Location}}</div>
                    <div class="finding-detail"><strong>值:</strong> <span class="finding-value">{{$finding.Value}}</span></div>
                </div>
                {{end}}
            </div>
            {{end}}
            
            {{if .MediumFindings}}
            <div class="severity-section severity-medium">
                <h2>🟡 中危发现 ({{len .MediumFindings}})</h2>
                {{range $index, $finding := .MediumFindings}}
                <div class="finding-card">
                    <h3>{{add $index 1}}. {{$finding.Type}}</h3>
                    <div class="finding-detail"><strong>来源URL:</strong> {{$finding.SourceURL}}</div>
                    <div class="finding-detail"><strong>位置:</strong> {{$finding.Location}}</div>
                    <div class="finding-detail"><strong>值:</strong> <span class="finding-value">{{$finding.Value}}</span></div>
                </div>
                {{end}}
            </div>
            {{end}}
            
            {{if .LowFindings}}
            <div class="severity-section severity-low">
                <h2>🟢 低危发现 ({{len .LowFindings}})</h2>
                {{range $index, $finding := .LowFindings}}
                <div class="finding-card">
                    <h3>{{add $index 1}}. {{$finding.Type}}</h3>
                    <div class="finding-detail"><strong>来源URL:</strong> {{$finding.SourceURL}}</div>
                    <div class="finding-detail"><strong>位置:</strong> {{$finding.Location}}</div>
                    <div class="finding-detail"><strong>值:</strong> <span class="finding-value">{{$finding.Value}}</span></div>
                </div>
                {{end}}
            </div>
            {{end}}
        </div>
        
        <div class="footer">
            <p>GogoSpider v4.2 - 敏感信息统一管理系统</p>
            <p>报告生成时间: {{.GeneratedTime}}</p>
        </div>
    </div>
</body>
</html>`
	
	t := template.Must(template.New("report").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(tmpl))
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	stats := sim.GetStatistics()
	severityStats := stats["by_severity"].(map[string]int)
	
	data := map[string]interface{}{
		"TargetDomain":   sim.targetDomain,
		"GeneratedTime":  time.Now().Format("2006-01-02 15:04:05"),
		"TotalFindings":  len(sim.uniqueFindings),
		"HighCount":      severityStats["HIGH"],
		"MediumCount":    severityStats["MEDIUM"],
		"LowCount":       severityStats["LOW"],
		"AffectedURLs":   stats["affected_urls_count"],
		"HighFindings":   sim.getFindingsBySeverity("HIGH"),
		"MediumFindings": sim.getFindingsBySeverity("MEDIUM"),
		"LowFindings":    sim.getFindingsBySeverity("LOW"),
	}
	
	return t.Execute(file, data)
}

// ExportSummary 导出摘要报告
func (sim *SensitiveInfoManager) ExportSummary() error {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()
	
	filename := filepath.Join(sim.outputDir, sim.baseFilename+"_summary.txt")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	stats := sim.GetStatistics()
	severityStats := stats["by_severity"].(map[string]int)
	typeStats := stats["by_type"].(map[string]int)
	
	file.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	file.WriteString("║                  敏感信息检测摘要报告                          ║\n")
	file.WriteString("╚══════════════════════════════════════════════════════════════╝\n\n")
	
	file.WriteString(fmt.Sprintf("目标域名: %s\n", sim.targetDomain))
	file.WriteString(fmt.Sprintf("扫描时间: %s\n", stats["scan_start_time"]))
	file.WriteString(fmt.Sprintf("发现总数: %d（已去重）\n\n", len(sim.uniqueFindings)))
	
	file.WriteString("【严重程度分布】\n")
	file.WriteString(fmt.Sprintf("  🔴 高危:  %d\n", severityStats["HIGH"]))
	file.WriteString(fmt.Sprintf("  🟡 中危:  %d\n", severityStats["MEDIUM"]))
	file.WriteString(fmt.Sprintf("  🟢 低危:  %d\n\n", severityStats["LOW"]))
	
	file.WriteString("【风险评估】\n")
	if severityStats["HIGH"] > 0 {
		file.WriteString("  ⚠️  存在高危敏感信息泄露，建议立即修复！\n")
	} else if severityStats["MEDIUM"] > 0 {
		file.WriteString("  ⚡ 存在中危敏感信息泄露，建议及时处理\n")
	} else if severityStats["LOW"] > 0 {
		file.WriteString("  ✅ 仅存在低危敏感信息，风险较低\n")
	} else {
		file.WriteString("  ✅ 未发现敏感信息泄露\n")
	}
	file.WriteString("\n")
	
	file.WriteString("【类型分布 TOP 10】\n")
	type typeStat struct {
		Type  string
		Count int
	}
	var typeList []typeStat
	for t, c := range typeStats {
		typeList = append(typeList, typeStat{Type: t, Count: c})
	}
	sort.Slice(typeList, func(i, j int) bool {
		return typeList[i].Count > typeList[j].Count
	})
	
	limit := 10
	if len(typeList) < limit {
		limit = len(typeList)
	}
	for i := 0; i < limit; i++ {
		file.WriteString(fmt.Sprintf("  %2d. %-30s: %d\n", i+1, typeList[i].Type, typeList[i].Count))
	}
	file.WriteString("\n")
	
	file.WriteString("【受影响URL统计】\n")
	file.WriteString(fmt.Sprintf("  受影响URL总数: %d\n", stats["affected_urls_count"]))
	
	mostAffected := stats["most_affected_urls"].([]map[string]interface{})
	if len(mostAffected) > 0 {
		file.WriteString("  受影响最多的URL:\n")
		for i, urlStat := range mostAffected {
			file.WriteString(fmt.Sprintf("    %d. %s (%d 项)\n", i+1, urlStat["url"], urlStat["count"]))
		}
	}
	file.WriteString("\n")
	
	file.WriteString("【详细报告文件】\n")
	file.WriteString(fmt.Sprintf("  📄 文本报告: %s.txt\n", sim.baseFilename))
	file.WriteString(fmt.Sprintf("  📊 JSON报告: %s.json\n", sim.baseFilename))
	file.WriteString(fmt.Sprintf("  📈 CSV报告:  %s.csv\n", sim.baseFilename))
	file.WriteString(fmt.Sprintf("  🌐 HTML报告: %s.html\n", sim.baseFilename))
	file.WriteString("\n")
	
	file.WriteString("══════════════════════════════════════════════════════════════\n")
	file.WriteString("报告生成时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n")
	file.WriteString("══════════════════════════════════════════════════════════════\n")
	
	return nil
}

// getFindingsBySeverity 获取指定严重程度的发现
func (sim *SensitiveInfoManager) getFindingsBySeverity(severity string) []*SensitiveInfo {
	findings := make([]*SensitiveInfo, 0)
	for _, finding := range sim.uniqueFindings {
		if finding.Severity == severity {
			findings = append(findings, finding)
		}
	}
	
	// 按类型排序
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Type < findings[j].Type
	})
	
	return findings
}

// printExportSummary 打印导出摘要
func (sim *SensitiveInfoManager) printExportSummary() {
	stats := sim.GetStatistics()
	severityStats := stats["by_severity"].(map[string]int)
	
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║          ✅ 敏感信息报告已统一导出                            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n📊 统计概览:\n")
	fmt.Printf("  发现总数: %d（已去重）\n", len(sim.uniqueFindings))
	fmt.Printf("  高危: %d  |  中危: %d  |  低危: %d\n",
		severityStats["HIGH"], severityStats["MEDIUM"], severityStats["LOW"])
	fmt.Printf("  受影响URL: %d\n", stats["affected_urls_count"])
	
	fmt.Printf("\n📁 导出文件:\n")
	fmt.Printf("  📄 %s.txt        - 详细文本报告\n", sim.baseFilename)
	fmt.Printf("  📊 %s.json       - 结构化JSON数据\n", sim.baseFilename)
	fmt.Printf("  📈 %s.csv        - Excel兼容表格\n", sim.baseFilename)
	fmt.Printf("  🌐 %s.html       - 可视化HTML报告\n", sim.baseFilename)
	fmt.Printf("  📋 %s_summary.txt - 快速摘要\n", sim.baseFilename)
	
	if severityStats["HIGH"] > 0 {
		fmt.Printf("\n⚠️  风险提示: 发现 %d 项高危敏感信息，请立即查看报告并处理！\n", severityStats["HIGH"])
	}
	
	fmt.Println("\n" + strings.Repeat("═", 64))
}

