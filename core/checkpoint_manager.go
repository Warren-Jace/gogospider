package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CheckpointManager 断点管理器
type CheckpointManager struct {
	checkpointDir  string
	saveInterval   time.Duration
	autoSave       bool
	currentState   *CrawlState
	mutex          sync.RWMutex
	stopChan       chan struct{}
	saveTicker     *time.Ticker
}

// CrawlState 爬取状态
type CrawlState struct {
	// 基本信息
	TaskID         string                 `json:"task_id"`
	TargetURL      string                 `json:"target_url"`
	StartTime      time.Time              `json:"start_time"`
	LastUpdateTime time.Time              `json:"last_update_time"`
	Status         string                 `json:"status"` // running, paused, completed, failed
	
	// 进度信息
	CurrentDepth   int                    `json:"current_depth"`
	MaxDepth       int                    `json:"max_depth"`
	TotalCrawled   int                    `json:"total_crawled"`
	TotalFailed    int                    `json:"total_failed"`
	
	// URL队列
	VisitedURLs    map[string]bool        `json:"visited_urls"`
	PendingURLs    []string               `json:"pending_urls"`
	FailedURLs     map[string]string      `json:"failed_urls"` // URL -> Error
	
	// 结果数据
	DiscoveredURLs []string               `json:"discovered_urls"`
	DiscoveredForms []FormCheckpoint      `json:"discovered_forms"`
	DiscoveredAPIs []string               `json:"discovered_apis"`
	
	// 配置信息
	Config         map[string]interface{} `json:"config"`
	
	// 性能统计
	Statistics     map[string]interface{} `json:"statistics"`
	
	// 自定义数据
	CustomData     map[string]interface{} `json:"custom_data,omitempty"`
}

// FormCheckpoint 表单检查点
type FormCheckpoint struct {
	Action string            `json:"action"`
	Method string            `json:"method"`
	Fields map[string]string `json:"fields"`
}

// NewCheckpointManager 创建断点管理器
func NewCheckpointManager(checkpointDir string, saveInterval time.Duration) *CheckpointManager {
	// 创建检查点目录
	if checkpointDir == "" {
		checkpointDir = "./checkpoints"
	}
	
	os.MkdirAll(checkpointDir, 0755)
	
	return &CheckpointManager{
		checkpointDir: checkpointDir,
		saveInterval:  saveInterval,
		autoSave:      false,
		stopChan:      make(chan struct{}),
	}
}

// InitState 初始化状态
func (cm *CheckpointManager) InitState(taskID, targetURL string, maxDepth int) *CrawlState {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	cm.currentState = &CrawlState{
		TaskID:         taskID,
		TargetURL:      targetURL,
		StartTime:      time.Now(),
		LastUpdateTime: time.Now(),
		Status:         "running",
		CurrentDepth:   1,
		MaxDepth:       maxDepth,
		TotalCrawled:   0,
		TotalFailed:    0,
		VisitedURLs:    make(map[string]bool),
		PendingURLs:    make([]string, 0),
		FailedURLs:     make(map[string]string),
		DiscoveredURLs: make([]string, 0),
		DiscoveredForms: make([]FormCheckpoint, 0),
		DiscoveredAPIs: make([]string, 0),
		Config:         make(map[string]interface{}),
		Statistics:     make(map[string]interface{}),
		CustomData:     make(map[string]interface{}),
	}
	
	return cm.currentState
}

// EnableAutoSave 启用自动保存
func (cm *CheckpointManager) EnableAutoSave() {
	if cm.autoSave {
		return
	}
	
	cm.autoSave = true
	cm.saveTicker = time.NewTicker(cm.saveInterval)
	
	go cm.autoSaveLoop()
	
	fmt.Printf("[断点续爬] 自动保存已启用，间隔: %v\n", cm.saveInterval)
}

// DisableAutoSave 禁用自动保存
func (cm *CheckpointManager) DisableAutoSave() {
	if !cm.autoSave {
		return
	}
	
	cm.autoSave = false
	
	if cm.saveTicker != nil {
		cm.saveTicker.Stop()
	}
	
	close(cm.stopChan)
	
	fmt.Println("[断点续爬] 自动保存已禁用")
}

// autoSaveLoop 自动保存循环
func (cm *CheckpointManager) autoSaveLoop() {
	for {
		select {
		case <-cm.saveTicker.C:
			err := cm.SaveCheckpoint()
			if err != nil {
				fmt.Printf("[断点续爬] 自动保存失败: %v\n", err)
			} else {
				fmt.Printf("[断点续爬] 自动保存成功 (%s)\n", time.Now().Format("15:04:05"))
			}
			
		case <-cm.stopChan:
			return
		}
	}
}

// UpdateState 更新状态
func (cm *CheckpointManager) UpdateState(updates map[string]interface{}) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return
	}
	
	// 更新时间
	cm.currentState.LastUpdateTime = time.Now()
	
	// 更新字段
	for key, value := range updates {
		switch key {
		case "status":
			if v, ok := value.(string); ok {
				cm.currentState.Status = v
			}
		case "current_depth":
			if v, ok := value.(int); ok {
				cm.currentState.CurrentDepth = v
			}
		case "total_crawled":
			if v, ok := value.(int); ok {
				cm.currentState.TotalCrawled = v
			}
		case "total_failed":
			if v, ok := value.(int); ok {
				cm.currentState.TotalFailed = v
			}
		}
	}
}

// AddVisitedURL 添加已访问URL
func (cm *CheckpointManager) AddVisitedURL(url string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return
	}
	
	cm.currentState.VisitedURLs[url] = true
	cm.currentState.TotalCrawled++
}

// AddPendingURL 添加待爬取URL
func (cm *CheckpointManager) AddPendingURL(url string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return
	}
	
	// 检查是否已访问
	if cm.currentState.VisitedURLs[url] {
		return
	}
	
	// 检查是否已在队列中
	for _, pending := range cm.currentState.PendingURLs {
		if pending == url {
			return
		}
	}
	
	cm.currentState.PendingURLs = append(cm.currentState.PendingURLs, url)
}

// AddPendingURLs 批量添加待爬取URL
func (cm *CheckpointManager) AddPendingURLs(urls []string) {
	for _, url := range urls {
		cm.AddPendingURL(url)
	}
}

// PopPendingURL 弹出一个待爬取URL
func (cm *CheckpointManager) PopPendingURL() (string, bool) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil || len(cm.currentState.PendingURLs) == 0 {
		return "", false
	}
	
	url := cm.currentState.PendingURLs[0]
	cm.currentState.PendingURLs = cm.currentState.PendingURLs[1:]
	
	return url, true
}

// AddFailedURL 添加失败URL
func (cm *CheckpointManager) AddFailedURL(url, errorMsg string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return
	}
	
	cm.currentState.FailedURLs[url] = errorMsg
	cm.currentState.TotalFailed++
}

// AddDiscoveredURL 添加发现的URL
func (cm *CheckpointManager) AddDiscoveredURL(url string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return
	}
	
	cm.currentState.DiscoveredURLs = append(cm.currentState.DiscoveredURLs, url)
}

// AddDiscoveredForm 添加发现的表单
func (cm *CheckpointManager) AddDiscoveredForm(form Form) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return
	}
	
	// 转换为检查点格式
	fields := make(map[string]string)
	for _, field := range form.Fields {
		fields[field.Name] = field.Value
	}
	
	checkpoint := FormCheckpoint{
		Action: form.Action,
		Method: form.Method,
		Fields: fields,
	}
	
	cm.currentState.DiscoveredForms = append(cm.currentState.DiscoveredForms, checkpoint)
}

// AddDiscoveredAPI 添加发现的API
func (cm *CheckpointManager) AddDiscoveredAPI(api string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return
	}
	
	cm.currentState.DiscoveredAPIs = append(cm.currentState.DiscoveredAPIs, api)
}

// SetConfig 设置配置
func (cm *CheckpointManager) SetConfig(key string, value interface{}) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return
	}
	
	cm.currentState.Config[key] = value
}

// SetStatistics 设置统计信息
func (cm *CheckpointManager) SetStatistics(stats map[string]interface{}) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return
	}
	
	cm.currentState.Statistics = stats
}

// SetCustomData 设置自定义数据
func (cm *CheckpointManager) SetCustomData(key string, value interface{}) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return
	}
	
	cm.currentState.CustomData[key] = value
}

// GetState 获取状态（副本）
func (cm *CheckpointManager) GetState() *CrawlState {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	if cm.currentState == nil {
		return nil
	}
	
	// 返回副本
	stateCopy := *cm.currentState
	return &stateCopy
}

// SaveCheckpoint 保存检查点
func (cm *CheckpointManager) SaveCheckpoint() error {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	if cm.currentState == nil {
		return fmt.Errorf("状态未初始化")
	}
	
	// 更新最后更新时间
	cm.currentState.LastUpdateTime = time.Now()
	
	// 序列化为JSON
	data, err := json.MarshalIndent(cm.currentState, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %v", err)
	}
	
	// 生成文件名
	filename := filepath.Join(cm.checkpointDir, 
		fmt.Sprintf("%s_checkpoint.json", cm.currentState.TaskID))
	
	// 写入临时文件
	tempFile := filename + ".tmp"
	err = ioutil.WriteFile(tempFile, data, 0644)
	if err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}
	
	// 原子替换
	err = os.Rename(tempFile, filename)
	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("重命名失败: %v", err)
	}
	
	return nil
}

// LoadCheckpoint 加载检查点
func (cm *CheckpointManager) LoadCheckpoint(taskID string) (*CrawlState, error) {
	filename := filepath.Join(cm.checkpointDir, 
		fmt.Sprintf("%s_checkpoint.json", taskID))
	
	// 检查文件是否存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("检查点文件不存在: %s", filename)
	}
	
	// 读取文件
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}
	
	// 反序列化
	var state CrawlState
	err = json.Unmarshal(data, &state)
	if err != nil {
		return nil, fmt.Errorf("解析失败: %v", err)
	}
	
	cm.mutex.Lock()
	cm.currentState = &state
	cm.mutex.Unlock()
	
	fmt.Printf("[断点续爬] 检查点已加载\n")
	fmt.Printf("  任务ID: %s\n", state.TaskID)
	fmt.Printf("  目标URL: %s\n", state.TargetURL)
	fmt.Printf("  开始时间: %s\n", state.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  最后更新: %s\n", state.LastUpdateTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  状态: %s\n", state.Status)
	fmt.Printf("  当前深度: %d/%d\n", state.CurrentDepth, state.MaxDepth)
	fmt.Printf("  已爬取: %d, 待爬取: %d, 失败: %d\n", 
		state.TotalCrawled, len(state.PendingURLs), state.TotalFailed)
	
	return &state, nil
}

// ListCheckpoints 列出所有检查点
func (cm *CheckpointManager) ListCheckpoints() ([]string, error) {
	files, err := ioutil.ReadDir(cm.checkpointDir)
	if err != nil {
		return nil, err
	}
	
	checkpoints := make([]string, 0)
	
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), "_checkpoint.json") {
			// 提取TaskID
			taskID := strings.TrimSuffix(file.Name(), "_checkpoint.json")
			checkpoints = append(checkpoints, taskID)
		}
	}
	
	return checkpoints, nil
}

// DeleteCheckpoint 删除检查点
func (cm *CheckpointManager) DeleteCheckpoint(taskID string) error {
	filename := filepath.Join(cm.checkpointDir, 
		fmt.Sprintf("%s_checkpoint.json", taskID))
	
	err := os.Remove(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	
	fmt.Printf("[断点续爬] 检查点已删除: %s\n", taskID)
	return nil
}

// GetProgress 获取进度百分比
func (cm *CheckpointManager) GetProgress() float64 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	if cm.currentState == nil {
		return 0
	}
	
	total := cm.currentState.TotalCrawled + len(cm.currentState.PendingURLs)
	if total == 0 {
		return 0
	}
	
	return float64(cm.currentState.TotalCrawled) / float64(total) * 100
}

// IsCompleted 是否完成
func (cm *CheckpointManager) IsCompleted() bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	if cm.currentState == nil {
		return false
	}
	
	return cm.currentState.Status == "completed"
}

// Pause 暂停
func (cm *CheckpointManager) Pause() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return fmt.Errorf("状态未初始化")
	}
	
	cm.currentState.Status = "paused"
	cm.currentState.LastUpdateTime = time.Now()
	
	fmt.Println("[断点续爬] 任务已暂停")
	
	// 保存检查点
	return cm.SaveCheckpoint()
}

// Resume 恢复
func (cm *CheckpointManager) Resume() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return fmt.Errorf("状态未初始化")
	}
	
	if cm.currentState.Status != "paused" {
		return fmt.Errorf("任务未暂停")
	}
	
	cm.currentState.Status = "running"
	cm.currentState.LastUpdateTime = time.Now()
	
	fmt.Println("[断点续爬] 任务已恢复")
	
	return nil
}

// Complete 标记完成
func (cm *CheckpointManager) Complete() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return fmt.Errorf("状态未初始化")
	}
	
	cm.currentState.Status = "completed"
	cm.currentState.LastUpdateTime = time.Now()
	
	fmt.Println("[断点续爬] 任务已完成")
	
	// 保存最终检查点
	return cm.SaveCheckpoint()
}

// Fail 标记失败
func (cm *CheckpointManager) Fail(reason string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentState == nil {
		return fmt.Errorf("状态未初始化")
	}
	
	cm.currentState.Status = "failed"
	cm.currentState.LastUpdateTime = time.Now()
	cm.currentState.CustomData["failure_reason"] = reason
	
	fmt.Printf("[断点续爬] 任务失败: %s\n", reason)
	
	// 保存检查点
	return cm.SaveCheckpoint()
}

// PrintProgress 打印进度
func (cm *CheckpointManager) PrintProgress() {
	state := cm.GetState()
	if state == nil {
		return
	}
	
	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("【爬取进度】")
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("  任务ID: %s\n", state.TaskID)
	fmt.Printf("  目标URL: %s\n", state.TargetURL)
	fmt.Printf("  状态: %s\n", state.Status)
	fmt.Printf("  深度: %d/%d\n", state.CurrentDepth, state.MaxDepth)
	fmt.Printf("  已爬取: %d\n", state.TotalCrawled)
	fmt.Printf("  待爬取: %d\n", len(state.PendingURLs))
	fmt.Printf("  失败: %d\n", state.TotalFailed)
	fmt.Printf("  进度: %.1f%%\n", cm.GetProgress())
	
	// 时间统计
	duration := time.Since(state.StartTime)
	fmt.Printf("  耗时: %s\n", duration.Round(time.Second))
	
	if state.TotalCrawled > 0 {
		avgTime := duration / time.Duration(state.TotalCrawled)
		fmt.Printf("  平均速度: %.1f URL/秒\n", 1.0/avgTime.Seconds())
	}
	
	// 发现统计
	fmt.Printf("\n  发现URL: %d\n", len(state.DiscoveredURLs))
	fmt.Printf("  发现表单: %d\n", len(state.DiscoveredForms))
	fmt.Printf("  发现API: %d\n", len(state.DiscoveredAPIs))
	
	fmt.Println(strings.Repeat("═", 60))
}

// ExportResults 导出结果
func (cm *CheckpointManager) ExportResults(filename string) error {
	state := cm.GetState()
	if state == nil {
		return fmt.Errorf("状态未初始化")
	}
	
	// 构建导出数据
	export := map[string]interface{}{
		"task_id":         state.TaskID,
		"target_url":      state.TargetURL,
		"start_time":      state.StartTime,
		"end_time":        state.LastUpdateTime,
		"duration":        state.LastUpdateTime.Sub(state.StartTime).String(),
		"status":          state.Status,
		"total_crawled":   state.TotalCrawled,
		"total_failed":    state.TotalFailed,
		"discovered_urls": state.DiscoveredURLs,
		"discovered_forms": state.DiscoveredForms,
		"discovered_apis": state.DiscoveredAPIs,
		"statistics":      state.Statistics,
	}
	
	// 序列化
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return err
	}
	
	// 写入文件
	return ioutil.WriteFile(filename, data, 0644)
}
