package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task 爬取任务
type Task struct {
	URL    string
	Depth  int
	Parent string
}

// WorkerPool 工作池管理器
type WorkerPool struct {
	workerCount int
	taskQueue   chan Task
	resultChan  chan *Result
	errorChan   chan error
	wg          sync.WaitGroup
	collectWg   sync.WaitGroup // 新增：用于等待collectResults goroutine
	ctx         context.Context
	cancel      context.CancelFunc

	// 统计信息
	totalTasks     int
	completedTasks int
	failedTasks    int
	mutex          sync.Mutex

	// 速率限制
	rateLimiter *time.Ticker
	maxQPS      int // 每秒最大请求数

	// 结果收集
	results      []*Result  // 新增：内部存储结果
	resultsMutex sync.Mutex // 新增：结果锁
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workerCount int, maxQPS int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workerCount: workerCount,
		taskQueue:   make(chan Task, workerCount*10), // 缓冲队列
		resultChan:  make(chan *Result, workerCount*2),
		errorChan:   make(chan error, workerCount),
		ctx:         ctx,
		cancel:      cancel,
		maxQPS:      maxQPS,
		rateLimiter: time.NewTicker(time.Second / time.Duration(maxQPS)),
		results:     make([]*Result, 0), // 初始化结果切片
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start(workerFunc func(task Task) (*Result, error)) {
	// 启动worker协程
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i, workerFunc)
	}

	// 启动结果收集协程（使用collectWg等待）
	wp.collectWg.Add(1)
	go wp.collectResults()
}

// worker 工作协程
func (wp *WorkerPool) worker(id int, workerFunc func(task Task) (*Result, error)) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			return

		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}

			// 速率限制
			<-wp.rateLimiter.C

			// 执行任务
			result, err := workerFunc(task)

			wp.mutex.Lock()
			wp.completedTasks++
			if err != nil {
				wp.failedTasks++
				wp.errorChan <- fmt.Errorf("worker %d: %v", id, err)
			} else if result != nil {
				wp.resultChan <- result
			}
			wp.mutex.Unlock()
		}
	}
}

// collectResults 收集结果
func (wp *WorkerPool) collectResults() {
	defer wp.collectWg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			// context取消，停止收集
			return

		case result, ok := <-wp.resultChan:
			if !ok {
				// resultChan已关闭，停止收集
				return
			}
			// 存储结果
			wp.resultsMutex.Lock()
			wp.results = append(wp.results, result)
			wp.resultsMutex.Unlock()

		case err, ok := <-wp.errorChan:
			if !ok {
				// errorChan已关闭
				return
			}
			// 处理错误（只记录，不中断）
			if err != nil {
				fmt.Printf("  工作池错误: %v\n", err)
			}
		}
	}
}

// Submit 提交任务
func (wp *WorkerPool) Submit(task Task) error {
	select {
	case <-wp.ctx.Done():
		return fmt.Errorf("工作池已关闭")
	case wp.taskQueue <- task:
		wp.mutex.Lock()
		wp.totalTasks++
		wp.mutex.Unlock()
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("提交任务超时")
	}
}

// Wait 等待所有任务完成
func (wp *WorkerPool) Wait() {
	// 关闭任务队列（不再接受新任务）
	close(wp.taskQueue)

	// 等待所有worker完成
	wp.wg.Wait()

	// worker完成后，关闭结果和错误channel
	// 这样collectResults能够正常退出
	close(wp.resultChan)
	close(wp.errorChan)

	// 等待结果收集完成
	wp.collectWg.Wait()
}

// Stop 停止工作池（紧急停止，不等待任务完成）
func (wp *WorkerPool) Stop() {
	// 取消context，通知所有goroutine停止
	wp.cancel()

	// 停止速率限制器
	if wp.rateLimiter != nil {
		wp.rateLimiter.Stop()
	}

	// 注意：channel已经在Wait()中关闭了
	// 这里只需要等待goroutine退出
	wp.wg.Wait()
	wp.collectWg.Wait()
}

// GetStats 获取统计信息
func (wp *WorkerPool) GetStats() map[string]int {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	return map[string]int{
		"total":     wp.totalTasks,
		"completed": wp.completedTasks,
		"failed":    wp.failedTasks,
		"pending":   wp.totalTasks - wp.completedTasks,
	}
}

// GetProgress 获取进度百分比
func (wp *WorkerPool) GetProgress() float64 {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	if wp.totalTasks == 0 {
		return 0
	}

	return float64(wp.completedTasks) / float64(wp.totalTasks) * 100
}

// GetResults 获取所有结果（在Wait()之后调用）
func (wp *WorkerPool) GetResults() []*Result {
	wp.resultsMutex.Lock()
	defer wp.resultsMutex.Unlock()

	// 返回结果副本
	results := make([]*Result, len(wp.results))
	copy(results, wp.results)
	return results
}
