package core

import (
	"strings"
	"sync"
)

// StringPool 字符串池（减少内存分配）
type StringPool struct {
	pool sync.Pool
}

// NewStringPool 创建字符串池
func NewStringPool() *StringPool {
	return &StringPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(strings.Builder)
			},
		},
	}
}

// Get 获取一个strings.Builder
func (sp *StringPool) Get() *strings.Builder {
	sb := sp.pool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// Put 归还strings.Builder
func (sp *StringPool) Put(sb *strings.Builder) {
	if sb.Cap() > 1024*1024 { // 如果容量超过1MB，不归还
		return
	}
	sp.pool.Put(sb)
}

// BuildString 构建字符串并自动归还builder
func (sp *StringPool) BuildString(fn func(*strings.Builder)) string {
	sb := sp.Get()
	defer sp.Put(sb)
	fn(sb)
	return sb.String()
}

// ByteSlicePool 字节切片池
type ByteSlicePool struct {
	pool sync.Pool
	size int
}

// NewByteSlicePool 创建字节切片池
func NewByteSlicePool(size int) *ByteSlicePool {
	return &ByteSlicePool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, size)
				return &buf
			},
		},
	}
}

// Get 获取字节切片
func (bsp *ByteSlicePool) Get() []byte {
	return *bsp.pool.Get().(*[]byte)
}

// Put 归还字节切片
func (bsp *ByteSlicePool) Put(buf []byte) {
	if cap(buf) != bsp.size {
		return // 大小不匹配，不归还
	}
	bsp.pool.Put(&buf)
}

// ResultPool 结果对象池
type ResultPool struct {
	pool sync.Pool
}

// NewResultPool 创建结果池
func NewResultPool() *ResultPool {
	return &ResultPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Result{
					Links:        make([]string, 0, 10),
					Assets:       make([]string, 0, 5),
					Forms:        make([]Form, 0, 2),
					APIs:         make([]string, 0, 5),
					POSTRequests: make([]POSTRequest, 0, 2),
					Headers:      make(map[string]string, 5),
				}
			},
		},
	}
}

// Get 获取Result对象
func (rp *ResultPool) Get() *Result {
	result := rp.pool.Get().(*Result)
	// 重置字段
	result.URL = ""
	result.StatusCode = 0
	result.ContentType = ""
	result.HTMLContent = ""
	result.Links = result.Links[:0]
	result.Assets = result.Assets[:0]
	result.Forms = result.Forms[:0]
	result.APIs = result.APIs[:0]
	result.POSTRequests = result.POSTRequests[:0]
	for k := range result.Headers {
		delete(result.Headers, k)
	}
	return result
}

// Put 归还Result对象
func (rp *ResultPool) Put(result *Result) {
	// 如果切片容量过大，不归还
	if cap(result.Links) > 1000 || cap(result.Assets) > 1000 {
		return
	}
	rp.pool.Put(result)
}

// 全局对象池
var (
	GlobalStringPool     *StringPool
	GlobalByteSlicePool  *ByteSlicePool
	GlobalResultPool     *ResultPool
)

func init() {
	GlobalStringPool = NewStringPool()
	GlobalByteSlicePool = NewByteSlicePool(4096) // 4KB缓冲
	GlobalResultPool = NewResultPool()
}

