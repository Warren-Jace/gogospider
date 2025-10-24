package improvements

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ==========================================
// 示例1: 优雅关闭和资源管理
// ==========================================

// ImprovedSpider 改进的爬虫结构
type ImprovedSpider struct {
	// 现有字段...
	
	// 新增：资源管理
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	done   chan struct{}
	
	// 新增：错误收集
	errors []error
	errMux sync.Mutex
}

// NewImprovedSpider 创建改进的爬虫实例
func NewImprovedSpider() *ImprovedSpider {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ImprovedSpider{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
		errors: make([]error, 0),
	}
}

// Start 开始爬取（改进版）
func (s *ImprovedSpider) Start(targetURL string) error {
	// 确保资源清理
	defer s.cleanup()
	
	// 使用 context 控制生命周期
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}
	
	// 启动工作协程
	s.wg.Add(1)
	go s.worker()
	
	// 等待完成或超时
	timeout := time.After(5 * time.Minute)
	select {
	case <-s.done:
		fmt.Println("爬取完成")
	case <-timeout:
		fmt.Println("爬取超时")
		s.Stop()
	case <-s.ctx.Done():
		fmt.Println("爬取被取消")
	}
	
	return nil
}

// Stop 优雅停止
func (s *ImprovedSpider) Stop() {
	fmt.Println("正在停止爬虫...")
	s.cancel() // 取消所有操作
}

// Close 实现 io.Closer 接口
func (s *ImprovedSpider) Close() error {
	s.Stop()
	s.wg.Wait()
	return nil
}

// cleanup 清理资源
func (s *ImprovedSpider) cleanup() {
	fmt.Println("清理资源...")
	s.wg.Wait()
	close(s.done)
	
	// 输出错误统计
	if len(s.errors) > 0 {
		fmt.Printf("爬取过程中发生 %d 个错误\n", len(s.errors))
	}
}

// worker 工作协程示例
func (s *ImprovedSpider) worker() {
	defer s.wg.Done()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// 执行爬取任务
			if err := s.doWork(); err != nil {
				s.recordError(err)
			}
			
			// 模拟工作完成
			time.Sleep(100 * time.Millisecond)
			return
		}
	}
}

// doWork 执行实际工作
func (s *ImprovedSpider) doWork() error {
	// 检查 context
	if err := s.ctx.Err(); err != nil {
		return err
	}
	
	// 模拟工作
	fmt.Println("执行爬取任务...")
	return nil
}

// recordError 记录错误（线程安全）
func (s *ImprovedSpider) recordError(err error) {
	s.errMux.Lock()
	defer s.errMux.Unlock()
	s.errors = append(s.errors, err)
}

// GetErrors 获取所有错误
func (s *ImprovedSpider) GetErrors() []error {
	s.errMux.Lock()
	defer s.errMux.Unlock()
	return append([]error{}, s.errors...) // 返回副本
}

// ==========================================
// 使用示例
// ==========================================

func ExampleUsage() {
	spider := NewImprovedSpider()
	defer spider.Close() // 确保资源清理
	
	// 方式1: 正常使用
	if err := spider.Start("https://example.com"); err != nil {
		fmt.Printf("爬取失败: %v\n", err)
	}
	
	// 方式2: 带超时控制
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- spider.Start("https://example.com")
	}()
	
	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("爬取失败: %v\n", err)
		}
	case <-ctx.Done():
		spider.Stop()
		fmt.Println("超时，已停止爬虫")
	}
	
	// 获取错误信息
	errors := spider.GetErrors()
	for _, err := range errors {
		fmt.Printf("错误: %v\n", err)
	}
}

