// 【快速修复补丁】- 提升URL收集效果
// 使用方法：
// 1. 备份原文件
// 2. 应用本补丁中的修改
// 3. 重新编译测试

package main

// ================================================================
// 修复1：提高每层URL数量限制（最简单最有效）
// 位置：core/spider.go 第1607-1610行
// ================================================================

// ❌ 旧代码（第1607-1610行）
/*
tasksToSubmit = append(tasksToSubmit, link)

// 每层限制100个URL
if len(tasksToSubmit) >= 100 {
    break
}
*/

// ✅ 新代码（替换上述代码）
/*
tasksToSubmit = append(tasksToSubmit, link)

// 🔧 修复：提高每层URL限制（100→500），从配置读取
maxURLsPerLayer := 500
if s.config.SchedulingSettings.HybridConfig.MaxURLsPerLayer > 0 {
    maxURLsPerLayer = s.config.SchedulingSettings.HybridConfig.MaxURLsPerLayer
}

if len(tasksToSubmit) >= maxURLsPerLayer {
    s.logger.Info("达到本层URL上限",
        "limit", maxURLsPerLayer,
        "total_candidates", len(allLinks))
    break
}
*/

// ================================================================
// 修复2：确保使用新版URL验证器
// 位置：core/spider.go 第182行
// ================================================================

// ❌ 旧代码（第182行）
/*
urlValidator: NewURLValidator(),
*/

// ✅ 新代码（如果存在SmartURLValidatorCompat）
/*
urlValidator: NewSmartURLValidatorCompat(),  // 使用v2.0智能验证器
*/

// ================================================================
// 修复3：添加保存所有发现的URL的函数
// 位置：cmd/spider/main.go（新增函数）
// ================================================================

/*
// saveAllDiscoveredURLs 保存所有发现的URL（包括未爬取的）
func saveAllDiscoveredURLs(spider *core.Spider, baseFilename string) error {
	file, err := os.Create(baseFilename + "_all_discovered.txt")
	if err != nil {
		return err
	}
	defer file.Close()
	
	urlSet := make(map[string]bool)
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	
	// 1. 保存已爬取页面的URL
	results := spider.GetResults()
	for _, result := range results {
		if !urlSet[result.URL] {
			writer.WriteString(result.URL + "\n")
			urlSet[result.URL] = true
		}
		
		// 2. 保存所有发现的Links（包括未爬取的）
		for _, link := range result.Links {
			if !urlSet[link] {
				writer.WriteString(link + "\n")
				urlSet[link] = true
			}
		}
		
		// 3. 保存API端点
		for _, api := range result.APIs {
			if !urlSet[api] {
				writer.WriteString(api + "\n")
				urlSet[api] = true
			}
		}
		
		// 4. 保存表单action
		for _, form := range result.Forms {
			if form.Action != "" && !urlSet[form.Action] {
				writer.WriteString(form.Action + "\n")
				urlSet[form.Action] = true
			}
		}
	}
	
	// 5. 保存静态资源
	staticResources := spider.GetStaticResources()
	for _, img := range staticResources.Images {
		if !urlSet[img] {
			writer.WriteString(img + "\n")
			urlSet[img] = true
		}
	}
	for _, video := range staticResources.Videos {
		if !urlSet[video] {
			writer.WriteString(video + "\n")
			urlSet[video] = true
		}
	}
	for _, audio := range staticResources.Audios {
		if !urlSet[audio] {
			writer.WriteString(audio + "\n")
			urlSet[audio] = true
		}
	}
	for _, font := range staticResources.Fonts {
		if !urlSet[font] {
			writer.WriteString(font + "\n")
			urlSet[font] = true
		}
	}
	for _, doc := range staticResources.Documents {
		if !urlSet[doc] {
			writer.WriteString(doc + "\n")
			urlSet[doc] = true
		}
	}
	for _, archive := range staticResources.Archives {
		if !urlSet[archive] {
			writer.WriteString(archive + "\n")
			urlSet[archive] = true
		}
	}
	
	// 6. 保存外部链接
	externalLinks := spider.GetExternalLinks()
	for _, link := range externalLinks {
		if !urlSet[link] {
			writer.WriteString(link + "\n")
			urlSet[link] = true
		}
	}
	
	// 7. 保存特殊协议链接
	specialLinks := spider.GetSpecialProtocolLinks()
	for _, link := range specialLinks.Mailto {
		if !urlSet[link] {
			writer.WriteString(link + "\n")
			urlSet[link] = true
		}
	}
	for _, link := range specialLinks.Tel {
		if !urlSet[link] {
			writer.WriteString(link + "\n")
			urlSet[link] = true
		}
	}
	for _, link := range specialLinks.WebSocket {
		if !urlSet[link] {
			writer.WriteString(link + "\n")
			urlSet[link] = true
		}
	}
	
	fmt.Printf("  - %s_all_discovered.txt : %d 个URL（完整收集，包括静态资源和外部链接）\n", 
		baseFilename, len(urlSet))
	
	return nil
}
*/

// 然后在main函数中调用（约第616行，saveExcludedURLs之后）
/*
// 🆕 保存所有发现的URL（包括未爬取的静态资源和外部链接）
if err := saveAllDiscoveredURLs(spider, baseFilename); err != nil {
	log.Printf("保存所有发现的URL失败: %v", err)
}
*/

// ================================================================
// 修复4：改进域名判断逻辑
// 位置：cmd/spider/main.go 第687-717行
// ================================================================

// ✅ 改进后的isInTargetDomain函数
/*
func isInTargetDomain(urlStr, targetDomain string) bool {
	// 忽略特殊协议
	if strings.HasPrefix(urlStr, "mailto:") || 
	   strings.HasPrefix(urlStr, "tel:") ||
	   strings.HasPrefix(urlStr, "javascript:") ||
	   strings.HasPrefix(urlStr, "data:") {
		return false
	}
	
	// 解析URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	
	// 获取URL的域名（使用Hostname()自动去除端口）
	urlHost := parsedURL.Hostname()
	if urlHost == "" {
		// 相对路径URL，视为目标域名
		return true
	}
	
	// 清理目标域名（去除协议和端口）
	cleanTarget := strings.TrimPrefix(targetDomain, "http://")
	cleanTarget = strings.TrimPrefix(cleanTarget, "https://")
	cleanTarget = strings.Split(cleanTarget, ":")[0]
	
	// 完全匹配
	if urlHost == cleanTarget {
		return true
	}
	
	// 子域名匹配（例如：api.example.com 匹配 example.com）
	if strings.HasSuffix(urlHost, "."+cleanTarget) {
		return true
	}
	
	// 检查是否是主域名的父域名（例如：example.com 匹配 www.example.com）
	if strings.HasPrefix(cleanTarget, urlHost+".") {
		return true
	}
	
	return false
}
*/

// ================================================================
// 修复5：配置文件优化建议
// 位置：config.json
// ================================================================

/*
{
  "scheduling_settings": {
    "algorithm": "HYBRID",
    "hybrid_config": {
      "max_urls_per_layer": 1000,  // 🔧 提高每层URL限制
      "enable_adaptive_learning": true
    }
  },
  "scope_settings": {
    "enabled": true,
    "stay_in_domain": false,       // 🔧 允许收集域外URL
    "allow_subdomains": true,      // ✅ 允许子域名
    "allow_http": true,
    "allow_https": true,
    "exclude_extensions": [        // 🔧 减少排除的扩展名（只排除明显无用的）
      "jpg", "jpeg", "png", "gif", "ico", "svg", "webp",
      "woff", "woff2", "ttf", "eot", "otf",
      "mp4", "avi", "mov", "mp3", "wav"
    ]
  },
  "deduplication_settings": {
    "enable_smart_param_dedup": true,
    "enable_business_aware_filter": false,  // 🔧 临时关闭，减少误杀
    "enable_url_pattern_recognition": true
  }
}
*/

// ================================================================
// 实施步骤（按优先级）
// ================================================================

/*
优先级P0（立即修复，影响最大）：
1. 修复1：提高URL限制（100→500）
2. 修复2：确认/升级URL验证器

优先级P1（重要修复）：
3. 修复3：添加saveAllDiscoveredURLs函数
4. 修复5：优化配置文件

优先级P2（增强修复）：
5. 修复4：改进域名判断逻辑

测试验证：
6. 重新编译
7. 对比修复前后的URL数量
8. 检查是否有误杀
*/

// ================================================================
// 快速测试脚本
// ================================================================

/*
# 1. 备份原文件
copy core\spider.go core\spider.go.backup
copy cmd\spider\main.go cmd\spider\main.go.backup

# 2. 应用修复补丁（手动修改上述代码）

# 3. 重新编译
go build -o spider_fixed.exe cmd/spider/main.go

# 4. 对比测试
echo "=== 修复前 ==="
.\spider.exe -url http://example.com -depth 2 -config config.json

echo "=== 修复后 ==="
.\spider_fixed.exe -url http://example.com -depth 2 -config config.json

# 5. 对比URL数量
dir /b spider_*.txt | findstr urls.txt
*/

// ================================================================
// 预期效果
// ================================================================

/*
修复前：
- spider_example.com_xxx_urls.txt: 11个URL
- spider_example.com_xxx_all_urls.txt: 59个URL

修复后：
- spider_example.com_xxx_urls.txt: 100-200个URL
- spider_example.com_xxx_all_urls.txt: 300-400个URL
- spider_example.com_xxx_all_discovered.txt: 400+个URL（新增）

提升：20-40倍
*/

