package core

import (
	"context"
	"fmt"
	"time"
	
	"github.com/chromedp/chromedp"
)

// EventTrigger JavaScript事件触发器
type EventTrigger struct {
	triggerInterval time.Duration // 事件触发间隔
	waitAfterTrigger time.Duration // 触发后等待时间
	maxEvents       int           // 最大触发事件数
	enabledEvents   []string      // 启用的事件类型
}

// EventTriggerResult 事件触发结果
type EventTriggerResult struct {
	TriggeredElements int      // 触发的元素数量
	TriggeredEvents   int      // 触发的事件数量
	NewURLsFound      []string // 发现的新URL
	NewFormsFound     []Form   // 发现的新表单
	DOMChanges        int      // DOM变化次数
}

// NewEventTrigger 创建事件触发器
func NewEventTrigger() *EventTrigger {
	return &EventTrigger{
		triggerInterval:  100 * time.Millisecond, // 默认100ms间隔
		waitAfterTrigger: 500 * time.Millisecond, // 触发后等待500ms
		maxEvents:        100,                     // 最多触发100个事件
		enabledEvents: []string{
			"click",
			"mouseover",
			"mouseenter",
			"focus",
			"change",
		},
	}
}

// SetTriggerInterval 设置触发间隔
func (et *EventTrigger) SetTriggerInterval(interval time.Duration) {
	et.triggerInterval = interval
}

// SetWaitAfterTrigger 设置触发后等待时间
func (et *EventTrigger) SetWaitAfterTrigger(wait time.Duration) {
	et.waitAfterTrigger = wait
}

// SetMaxEvents 设置最大事件数
func (et *EventTrigger) SetMaxEvents(max int) {
	et.maxEvents = max
}

// TriggerEvents 触发页面上的所有事件
func (et *EventTrigger) TriggerEvents(ctx context.Context) (*EventTriggerResult, error) {
	result := &EventTriggerResult{
		NewURLsFound:  make([]string, 0),
		NewFormsFound: make([]Form, 0),
	}
	
	fmt.Println("  [事件触发] 开始自动触发页面事件...")
	
	// 1. 获取页面初始状态
	var initialHTML string
	err := chromedp.Run(ctx,
		chromedp.OuterHTML("html", &initialHTML),
	)
	if err != nil {
		return result, fmt.Errorf("获取初始HTML失败: %v", err)
	}
	
	// 2. 注入事件监听脚本
	err = et.injectEventListener(ctx)
	if err != nil {
		fmt.Printf("  [事件触发] 注入监听脚本失败: %v\n", err)
	}
	
	// 3. 触发点击事件
	clickCount, err := et.triggerClickEvents(ctx)
	if err != nil {
		fmt.Printf("  [事件触发] 触发点击事件出错: %v\n", err)
	} else {
		result.TriggeredElements += clickCount
		fmt.Printf("  [事件触发] 触发了 %d 个点击事件\n", clickCount)
	}
	
	// 4. 触发悬停事件
	hoverCount, err := et.triggerHoverEvents(ctx)
	if err != nil {
		fmt.Printf("  [事件触发] 触发悬停事件出错: %v\n", err)
	} else {
		result.TriggeredElements += hoverCount
		fmt.Printf("  [事件触发] 触发了 %d 个悬停事件\n", hoverCount)
	}
	
	// 5. 触发表单输入事件
	inputCount, err := et.triggerInputEvents(ctx)
	if err != nil {
		fmt.Printf("  [事件触发] 触发输入事件出错: %v\n", err)
	} else {
		result.TriggeredElements += inputCount
		fmt.Printf("  [事件触发] 触发了 %d 个输入事件\n", inputCount)
	}
	
	// 6. 等待DOM更新
	time.Sleep(et.waitAfterTrigger)
	
	// 7. 获取更新后的HTML
	var updatedHTML string
	err = chromedp.Run(ctx,
		chromedp.OuterHTML("html", &updatedHTML),
	)
	if err == nil && updatedHTML != initialHTML {
		result.DOMChanges = 1
		fmt.Println("  [事件触发] 检测到DOM变化")
	}
	
	// 8. 提取新发现的URL和表单
	newURLs, newForms := et.extractNewContent(ctx)
	result.NewURLsFound = newURLs
	result.NewFormsFound = newForms
	
	fmt.Printf("  [事件触发] 完成！发现 %d 个新URL, %d 个新表单\n", 
		len(newURLs), len(newForms))
	
	return result, nil
}

// injectEventListener 注入事件监听脚本
func (et *EventTrigger) injectEventListener(ctx context.Context) error {
	// JavaScript脚本：监听所有URL变化
	script := `
(function() {
    window.crawlergoURLs = window.crawlergoURLs || new Set();
    
    // 监听所有链接点击
    document.addEventListener('click', function(e) {
        var target = e.target;
        var href = target.href || target.getAttribute('href');
        if (href) {
            window.crawlergoURLs.add(href);
        }
    }, true);
    
    // 监听fetch和XHR
    var originalFetch = window.fetch;
    window.fetch = function() {
        var url = arguments[0];
        if (typeof url === 'string') {
            window.crawlergoURLs.add(url);
        }
        return originalFetch.apply(this, arguments);
    };
    
    var originalOpen = XMLHttpRequest.prototype.open;
    XMLHttpRequest.prototype.open = function(method, url) {
        window.crawlergoURLs.add(url);
        return originalOpen.apply(this, arguments);
    };
})();
`
	
	return chromedp.Run(ctx,
		chromedp.Evaluate(script, nil),
	)
}

// triggerClickEvents 触发点击事件
func (et *EventTrigger) triggerClickEvents(ctx context.Context) (int, error) {
	// JavaScript脚本：查找并点击所有可点击元素
	script := `
(function() {
    var selectors = [
        'button:not([disabled])',
        'a[href]:not([href^="javascript:"]):not([href^="mailto:"]):not([href^="#"])',
        'input[type="button"]:not([disabled])',
        'input[type="submit"]:not([disabled])',
        '[onclick]',
        '[role="button"]'
    ];
    
    var elements = [];
    selectors.forEach(function(selector) {
        var found = document.querySelectorAll(selector);
        Array.from(found).forEach(function(el) {
            if (elements.indexOf(el) === -1 && isVisible(el)) {
                elements.push(el);
            }
        });
    });
    
    function isVisible(el) {
        var style = window.getComputedStyle(el);
        return style.display !== 'none' && 
               style.visibility !== 'hidden' && 
               style.opacity !== '0' &&
               el.offsetWidth > 0 && 
               el.offsetHeight > 0;
    }
    
    var count = 0;
    var maxClicks = ` + fmt.Sprintf("%d", et.maxEvents) + `;
    
    for (var i = 0; i < elements.length && count < maxClicks; i++) {
        try {
            var element = elements[i];
            
            // 滚动到元素可见
            element.scrollIntoView({behavior: 'smooth', block: 'center'});
            
            // 触发点击
            var clickEvent = new MouseEvent('click', {
                bubbles: true,
                cancelable: true,
                view: window
            });
            element.dispatchEvent(clickEvent);
            
            count++;
            
        } catch (e) {
            // 忽略错误，继续下一个元素
        }
    }
    
    return count;
})();
`
	
	var clickCount int
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &clickCount),
	)
	
	if err != nil {
		return 0, err
	}
	
	// 等待DOM更新
	time.Sleep(et.waitAfterTrigger)
	
	return clickCount, nil
}

// triggerHoverEvents 触发悬停事件
func (et *EventTrigger) triggerHoverEvents(ctx context.Context) (int, error) {
	script := `
(function() {
    var selectors = [
        'a[href]',
        'button',
        '[onmouseover]',
        '[onmouseenter]',
        'nav a',
        '.menu a',
        '.dropdown'
    ];
    
    var elements = [];
    selectors.forEach(function(selector) {
        try {
            var found = document.querySelectorAll(selector);
            Array.from(found).forEach(function(el) {
                if (elements.indexOf(el) === -1) {
                    elements.push(el);
                }
            });
        } catch (e) {}
    });
    
    var count = 0;
    var maxHovers = 50;
    
    for (var i = 0; i < elements.length && count < maxHovers; i++) {
        try {
            var element = elements[i];
            
            var mouseoverEvent = new MouseEvent('mouseover', {
                bubbles: true,
                cancelable: true,
                view: window
            });
            element.dispatchEvent(mouseoverEvent);
            
            var mouseenterEvent = new MouseEvent('mouseenter', {
                bubbles: true,
                cancelable: true,
                view: window
            });
            element.dispatchEvent(mouseenterEvent);
            
            count++;
        } catch (e) {}
    }
    
    return count;
})();
`
	
	var hoverCount int
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &hoverCount),
	)
	
	if err != nil {
		return 0, err
	}
	
	// 短暂等待
	time.Sleep(200 * time.Millisecond)
	
	return hoverCount, nil
}

// triggerInputEvents 触发输入框事件
func (et *EventTrigger) triggerInputEvents(ctx context.Context) (int, error) {
	script := `
(function() {
    var inputs = document.querySelectorAll('input[type="text"], input[type="search"], textarea');
    var count = 0;
    
    Array.from(inputs).forEach(function(input) {
        try {
            // 聚焦
            input.focus();
            
            // 输入测试值
            input.value = 'crawlergo_test';
            
            // 触发事件
            var inputEvent = new Event('input', {bubbles: true});
            input.dispatchEvent(inputEvent);
            
            var changeEvent = new Event('change', {bubbles: true});
            input.dispatchEvent(changeEvent);
            
            count++;
        } catch (e) {}
    });
    
    return count;
})();
`
	
	var inputCount int
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &inputCount),
	)
	
	if err != nil {
		return 0, err
	}
	
	// 等待事件处理
	time.Sleep(300 * time.Millisecond)
	
	return inputCount, nil
}

// extractNewContent 提取事件触发后的新内容
func (et *EventTrigger) extractNewContent(ctx context.Context) ([]string, []Form) {
	urls := make([]string, 0)
	forms := make([]Form, 0)
	
	// 提取所有链接
	script := `
(function() {
    var links = [];
    var aElements = document.querySelectorAll('a[href]');
    
    Array.from(aElements).forEach(function(a) {
        var href = a.href;
        if (href && 
            !href.startsWith('javascript:') && 
            !href.startsWith('mailto:') &&
            !href.startsWith('tel:')) {
            links.push(href);
        }
    });
    
    // 从监听到的URL中获取
    if (window.crawlergoURLs) {
        window.crawlergoURLs.forEach(function(url) {
            if (links.indexOf(url) === -1) {
                links.push(url);
            }
        });
    }
    
    return links;
})();
`
	
	var extractedURLs []interface{}
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &extractedURLs),
	)
	
	if err == nil {
		for _, u := range extractedURLs {
			if urlStr, ok := u.(string); ok {
				urls = append(urls, urlStr)
			}
		}
	}
	
	// 提取表单
	formScript := `
(function() {
    var forms = [];
    var formElements = document.querySelectorAll('form');
    
    Array.from(formElements).forEach(function(form) {
        var formData = {
            action: form.action || window.location.href,
            method: form.method || 'GET',
            fields: []
        };
        
        var inputs = form.querySelectorAll('input, select, textarea');
        Array.from(inputs).forEach(function(input) {
            formData.fields.push({
                name: input.name || input.id || '',
                type: input.type || 'text',
                value: input.value || ''
            });
        });
        
        if (formData.fields.length > 0) {
            forms.push(formData);
        }
    });
    
    return forms;
})();
`
	
	var extractedForms []interface{}
	err = chromedp.Run(ctx,
		chromedp.Evaluate(formScript, &extractedForms),
	)
	
	if err == nil {
		for _, f := range extractedForms {
			if formMap, ok := f.(map[string]interface{}); ok {
				form := Form{
					Action: getString(formMap, "action"),
					Method: getString(formMap, "method"),
					Fields: make([]FormField, 0),
				}
				
				if fieldsData, ok := formMap["fields"].([]interface{}); ok {
					for _, fieldData := range fieldsData {
						if fieldMap, ok := fieldData.(map[string]interface{}); ok {
							field := FormField{
								Name:  getString(fieldMap, "name"),
								Type:  getString(fieldMap, "type"),
								Value: getString(fieldMap, "value"),
							}
							form.Fields = append(form.Fields, field)
						}
					}
				}
				
				if len(form.Fields) > 0 {
					forms = append(forms, form)
				}
			}
		}
	}
	
	return urls, forms
}

// TriggerInfiniteScroll 触发无限滚动
func (et *EventTrigger) TriggerInfiniteScroll(ctx context.Context) (int, error) {
	script := `
(function() {
    var scrollCount = 0;
    var maxScrolls = 5;
    var lastHeight = document.body.scrollHeight;
    
    function scrollDown() {
        window.scrollTo(0, document.body.scrollHeight);
        scrollCount++;
    }
    
    // 滚动到底部
    for (var i = 0; i < maxScrolls; i++) {
        scrollDown();
        
        // 等待一下（通过改变一个标记）
        var waited = false;
        setTimeout(function() { waited = true; }, 1000);
        
        // 检查高度变化
        var newHeight = document.body.scrollHeight;
        if (newHeight === lastHeight) {
            break; // 没有新内容，停止滚动
        }
        lastHeight = newHeight;
    }
    
    // 滚动回顶部
    window.scrollTo(0, 0);
    
    return scrollCount;
})();
`
	
	var scrollCount int
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &scrollCount),
	)
	
	if err != nil {
		return 0, err
	}
	
	// 等待加载
	time.Sleep(1 * time.Second)
	
	return scrollCount, nil
}

// MonitorDOMChanges 监控DOM变化
func (et *EventTrigger) MonitorDOMChanges(ctx context.Context) error {
	// 注入MutationObserver
	script := `
(function() {
    if (window.crawlergoObserver) {
        return; // 已经注入过
    }
    
    window.crawlergoNewURLs = new Set();
    
    // 创建观察器
    var observer = new MutationObserver(function(mutations) {
        mutations.forEach(function(mutation) {
            // 检查新添加的节点
            mutation.addedNodes.forEach(function(node) {
                if (node.nodeType === 1) { // Element节点
                    // 查找链接
                    var links = node.querySelectorAll ? node.querySelectorAll('a[href]') : [];
                    Array.from(links).forEach(function(a) {
                        if (a.href) {
                            window.crawlergoNewURLs.add(a.href);
                        }
                    });
                    
                    // 如果节点本身是链接
                    if (node.tagName === 'A' && node.href) {
                        window.crawlergoNewURLs.add(node.href);
                    }
                }
            });
        });
    });
    
    // 开始观察
    observer.observe(document.body, {
        childList: true,
        subtree: true,
        attributes: false
    });
    
    window.crawlergoObserver = observer;
})();
`
	
	return chromedp.Run(ctx,
		chromedp.Evaluate(script, nil),
	)
}

// GetMonitoredURLs 获取监控到的新URL
func (et *EventTrigger) GetMonitoredURLs(ctx context.Context) ([]string, error) {
	script := `
(function() {
    if (!window.crawlergoNewURLs) {
        return [];
    }
    return Array.from(window.crawlergoNewURLs);
})();
`
	
	var urls []interface{}
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &urls),
	)
	
	if err != nil {
		return []string{}, err
	}
	
	result := make([]string, 0)
	for _, u := range urls {
		if urlStr, ok := u.(string); ok {
			result = append(result, urlStr)
		}
	}
	
	return result, nil
}

// TriggerSelectChange 触发下拉框change事件
func (et *EventTrigger) TriggerSelectChange(ctx context.Context) (int, error) {
	script := `
(function() {
    var selects = document.querySelectorAll('select');
    var count = 0;
    
    Array.from(selects).forEach(function(select) {
        try {
            // 如果有选项，选择第一个非空选项
            if (select.options && select.options.length > 0) {
                for (var i = 0; i < select.options.length; i++) {
                    if (select.options[i].value) {
                        select.selectedIndex = i;
                        break;
                    }
                }
            }
            
            // 触发change事件
            var changeEvent = new Event('change', {bubbles: true});
            select.dispatchEvent(changeEvent);
            
            count++;
        } catch (e) {}
    });
    
    return count;
})();
`
	
	var selectCount int
	err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &selectCount),
	)
	
	if err != nil {
		return 0, err
	}
	
	time.Sleep(200 * time.Millisecond)
	
	return selectCount, nil
}

