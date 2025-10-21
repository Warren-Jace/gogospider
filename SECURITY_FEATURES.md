# 安全爬虫增强功能说明

## 概述

本增强版爬虫专门为安全专家设计，提供了全面的安全测试功能，能够发现更多隐藏路径和参数，并自动识别潜在的安全风险点。

## 新增功能

### 1. 增强参数发现机制

#### 多源参数发现
- **HTML表单参数**：从input、select、textarea等表单元素中提取参数
- **JavaScript参数**：从JS代码中提取API调用、变量赋值等参数
- **HTTP响应头参数**：从Set-Cookie、Location重定向等头部信息中提取参数
- **HTML注释参数**：从注释中发现开发者遗留的参数信息

#### 安全参数分析
自动识别高风险参数类型：
- **DANGEROUS**：可能存在严重漏洞的参数（如file、cmd、exec等）
- **FILE_INCLUSION**：文件包含漏洞相关参数
- **SQL_INJECTION**：SQL注入风险参数（如id、user、search等）
- **XSS**：跨站脚本风险参数（如message、comment、content等）
- **SECURITY**：敏感功能参数（如debug、admin、password等）

### 2. 隐藏路径发现

#### 系统文件发现
- **robots.txt**：分析robots文件中的Disallow路径
- **sitemap.xml**：从站点地图中提取所有URL
- **备份文件**：自动测试.bak、.old、.backup等备份文件
- **配置文件**：发现.htaccess、web.config、.env等配置文件

#### 管理路径发现
- **通用管理路径**：/admin、/manage、/control等
- **CMS管理路径**：/wp-admin、/drupal/admin等
- **数据库管理**：/phpmyadmin、/pma等
- **API管理**：/swagger、/api-docs等

#### 常见敏感路径
- **开发环境**：/dev、/test、/debug等
- **文件管理**：/upload、/files、/backup等
- **系统目录**：/system、/includes、/vendor等

### 3. 安全测试变体生成

#### 自动化Payload生成
为每个发现的参数自动生成安全测试变体：
- **SQL注入测试**：'、"、1' OR '1'='1等
- **XSS测试**：&lt;script&gt;alert(1)&lt;/script&gt;、&lt;img src=x onerror=alert(1)&gt;等
- **文件包含测试**：../../../etc/passwd、C:\\Windows\\System32等
- **命令注入测试**：; ls、| whoami、&& dir等

#### 参数模糊测试
- **常见参数名**：自动测试debug、admin、test等隐藏参数
- **参数数组测试**：param[]、param[0]等数组格式测试
- **HTTP参数污染**：重复参数名测试

### 4. 增强报告格式

#### 安全导向的输出结构
报告按安全测试优先级组织：

1. **安全发现**：高风险参数和潜在漏洞点
2. **隐藏路径发现**：敏感文件和管理路径
3. **POST表单参数**：重点标注，包含详细字段信息
4. **GET参数URL**：SQL注入和XSS测试点
5. **API端点**：认证绕过测试目标
6. **所有发现链接**：完整的攻击面

#### 清晰的参数标注
- `POST:URL|param1=value1&param2=value2` - POST请求参数
- `GET:URL?param=value` - GET请求参数  
- `API:URL` - API端点
- `SECURITY_PARAM: paramname - risk description` - 安全风险参数

## 使用方法

### 1. 快速安全扫描
```bash
# 使用安全配置进行扫描
go run cmd/spider/main.go -config=security_config.json
```

### 2. 自定义目标扫描
```bash
# 指定目标URL和深度
go run cmd/spider/main.go -url=http://target.com -depth=3 -deep=true
```

### 3. 使用预置脚本
```bash
# Windows环境
security_test.bat

# 手动编译
go build -o security_spider.exe cmd/spider/main.go
security_spider.exe -config=security_config.json
```

## 安全测试建议

### POST参数测试重点
1. **SQL注入测试**：对所有输入字段进行SQL注入测试
2. **XSS测试**：特别关注text、textarea类型字段
3. **文件上传测试**：检查file类型字段的安全性
4. **隐藏字段分析**：重点关注hidden类型字段的值

### GET参数测试重点  
1. **数字型参数**：测试SQL注入（如id、page等）
2. **字符串参数**：测试XSS和SQL注入（如search、name等）
3. **文件路径参数**：测试文件包含漏洞（如file、path等）
4. **重定向参数**：测试开放重定向（如redirect、return等）

### 隐藏路径利用
1. **配置文件**：查找数据库连接信息、API密钥等
2. **备份文件**：可能包含源码或敏感信息
3. **管理界面**：尝试弱密码或默认凭据
4. **调试页面**：可能泄露系统信息

### API安全测试
1. **认证绕过**：测试未授权访问
2. **参数注入**：API参数的各种注入测试
3. **HTTP方法测试**：尝试不同的HTTP方法
4. **版本枚举**：测试不同的API版本

## 报告文件说明

生成的报告文件按以下格式命名：
`spider_[domain]_[timestamp].txt`

报告包含七个主要部分：
1. **扫描统计**：总体发现数量
2. **安全发现**：高优先级安全问题
3. **隐藏路径**：敏感文件和目录
4. **POST表单**：详细的表单参数信息
5. **GET参数URL**：带参数的URL列表
6. **API端点**：发现的API接口
7. **所有链接**：完整的链接清单

## 配置选项

### 安全配置建议
- `MaxDepth: 4`：增加爬取深度发现更多内容
- `DeepCrawling: true`：启用深度爬取
- `SimilarityThreshold: 0.8`：降低去重阈值发现更多变体
- `EnableJSAnalysis: true`：启用JS分析发现隐藏API
- `DomainScope`：限制扫描范围避免扫描外部站点

### 性能调优
- `RequestDelay: 500ms`：平衡速度与隐蔽性
- `Parallelism: 5`：适度的并发数量
- 多个User-Agent轮换避免被检测

## 注意事项

1. **合法性**：仅在授权的系统上进行安全测试
2. **范围控制**：使用DomainScope限制扫描范围
3. **速率限制**：适当的RequestDelay避免对目标系统造成压力
4. **数据处理**：及时清理敏感的扫描结果
5. **误报处理**：人工验证自动发现的安全问题

## 扩展建议

1. **集成漏洞扫描器**：将发现的URL导入专业漏洞扫描工具
2. **自动化测试**：基于发现的参数编写自动化安全测试脚本
3. **持续监控**：定期运行发现新的攻击面
4. **结果去重**：多次扫描结果的智能合并
5. **风险评级**：根据发现的问题进行风险等级划分
