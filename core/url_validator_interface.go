package core

// URLValidatorInterface URL验证器接口
// 定义URL验证器必须实现的方法
type URLValidatorInterface interface {
	// IsValidBusinessURL 判断URL是否为有效的业务URL
	IsValidBusinessURL(rawURL string) bool
	
	// FilterURLs 批量过滤URL列表
	FilterURLs(urls []string) []string
}

// 确保类型实现了接口
var (
	_ URLValidatorInterface = (*URLValidator)(nil)
	_ URLValidatorInterface = (*SmartURLValidatorCompat)(nil)
)

