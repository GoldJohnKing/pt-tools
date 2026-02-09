package definitions

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"

	v2 "github.com/sunerpy/pt-tools/site/v2"
)

// ttgDriver 是 TTG 的自定义驱动
// 继承 NexusPHPDriver 的所有功能，但重写 GetTorrentDetail 以处理 TTG 特殊的免费时间格式
type ttgDriver struct {
	*v2.NexusPHPDriver
}

// TTGDefinition is the site definition for TTG (To The Glory)
var TTGDefinition = &v2.SiteDefinition{
	ID:             "ttg",
	Name:           "TTG",
	Aka:            []string{"TTG", "To The Glory"},
	Description:    "TTG (To The Glory) 综合性PT站点",
	Schema:         v2.SchemaNexusPHP,
	URLs:           []string{"https://totheglory.im/"},
	FaviconURL:     "https://totheglory.im/favicon.ico",
	TimezoneOffset: "+0800",
	UserInfo: &v2.UserInfoConfig{
		PickLast:     []string{"id"},
		RequestDelay: 500,
		Process: []v2.UserInfoProcess{
			{
				RequestConfig: v2.RequestConfig{
					URL:          "/index.php",
					ResponseType: "document",
				},
				Fields: []string{"id", "name", "uploaded", "downloaded", "ratio", "seeding", "leeching", "bonus", "messageCount"},
			},
			{
				RequestConfig: v2.RequestConfig{
					URL:          "/userdetails.php",
					ResponseType: "document",
				},
				Assertion: map[string]string{"id": "params.id"},
				Fields: []string{
					"joinTime",
					"levelName",
				},
			},
			{
				RequestConfig: v2.RequestConfig{
					URL:          "/mybonus.php",
					ResponseType: "document",
				},
				Fields: []string{"bonusPerHour"},
			},
		},
		Selectors: map[string]v2.FieldSelector{
			// User ID from index.php top navigation
			// HTML: 欢迎回来，<b><a href="https://totheglory.im/userdetails.php?id=151907">username</a></b>
			"id": {
				Selector: []string{
					"a[href*='userdetails.php']",
				},
				Attr:    "href",
				Filters: []v2.Filter{{Name: "querystring", Args: []any{"id"}}},
			},
			// Username from index.php top navigation
			// HTML: 欢迎回来，<b><a href="https://totheglory.im/userdetails.php?id=151907">username</a></b>
			"name": {
				Selector: []string{
					"a[href*='userdetails.php']",
				},
			},
			// Upload from index.php top navigation
			// HTML: <font color="green">上传量 : </font> <font color="black"><a href="..." title="4,400,128.37 MB">4.196 TB</a></font>
			"uploaded": {
				Selector: []string{
					"td.bottom",
				},
				Attr: "html",
				Filters: []v2.Filter{
					{Name: "regex", Args: []any{`上传量\s*:\s*</font>\s*<font[^>]*><a[^>]*>([\d.,]+\s*[KMGTP]?i?B)`}},
					{Name: "parseSize"},
				},
			},
			// Download from index.php top navigation
			// HTML: <font color="darkred">下载量 :</font> <font color="black"><a href="..." title="1,558,484.04 MB">1.486 TB</a></font>
			"downloaded": {
				Selector: []string{
					"td.bottom",
				},
				Attr: "html",
				Filters: []v2.Filter{
					{Name: "regex", Args: []any{`下载量\s*:\s*</font>\s*<font[^>]*><a[^>]*>([\d.,]+\s*[KMGTP]?i?B)`}},
					{Name: "parseSize"},
				},
			},
			// Ratio from index.php top navigation
			// HTML: <font color="1900D1">分享率 :</font> <font color="#000000">2.823</font>
			"ratio": {
				Selector: []string{
					"td.bottom",
				},
				Attr: "html",
				Filters: []v2.Filter{
					{Name: "regex", Args: []any{`分享率\s*:\s*</font>\s*<font[^>]*>([\d.,]+|∞|Inf)`}},
					{Name: "parseNumber"},
				},
			},
			// Seeding count from index.php top navigation
			// HTML: <img alt="做种中" .../><span class="smallfont">10</span>
			"seeding": {
				Selector: []string{
					"td.bottom",
				},
				Attr: "html",
				Filters: []v2.Filter{
					{Name: "regex", Args: []any{`做种中.*?smallfont[^>]*>(\d+)`}},
					{Name: "parseNumber"},
				},
			},
			// Leeching count from index.php top navigation
			// HTML: <img alt="下载中" .../><span class="smallfont">0</span>
			"leeching": {
				Selector: []string{
					"td.bottom",
				},
				Attr: "html",
				Filters: []v2.Filter{
					{Name: "regex", Args: []any{`下载中.*?smallfont[^>]*>(\d+)`}},
					{Name: "parseNumber"},
				},
			},
			// Bonus (积分) from index.php top navigation
			// HTML: 积分 : <a href="https://totheglory.im/mybonus.php">908728.22</a>
			"bonus": {
				Selector: []string{
					"td.bottom",
				},
				Attr: "html",
				Filters: []v2.Filter{
					{Name: "regex", Args: []any{`积分\s*:\s*<[^>]*>([\d,]+\.?\d*)`}},
					{Name: "parseNumber"},
				},
			},
			// BonusPerHour from mybonus.php
			// HTML: <tr><td class="rowhead">总计</td><td>27.64 分</td></tr>
			"bonusPerHour": {
				Selector: []string{
					"body",
				},
				Attr: "html",
				Filters: []v2.Filter{
					{Name: "regex", Args: []any{`总计</td>.*?([\d.,]+)\s*分`}},
					{Name: "parseNumber"},
				},
			},
			// Level name from userdetails.php
			// HTML: <tr><td class="rowhead">等级</td><td align="left">...PetaByte</td></tr>
			"levelName": {
				Selector: []string{
					"td.rowhead:contains('等级') + td",
					"td.rowhead:contains('等級') + td",
					"td.rowhead:contains('Class') + td",
				},
			},
			// Join date from userdetails.php
			// HTML: <tr><td class="rowhead">注册日期</td><td align="left">2019-09-03 23:02:35</td></tr>
			"joinTime": {
				Selector: []string{
					"td.rowhead:contains('注册日期') + td",
					"td.rowhead:contains('Join') + td",
				},
				Filters: []v2.Filter{
					{Name: "regex", Args: []any{`(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})`}},
					{Name: "parseTime"},
				},
			},
			// Message count from index.php top navigation
			// HTML: <a href="messages.php?action=viewmailbox&box=1">...</a> 96 (0 <b>新</b>)
			"messageCount": {
				Text: "0",
				Selector: []string{
					"a[href*='messages.php'][href*='viewmailbox']",
				},
				Filters: []v2.Filter{
					{Name: "parentText"},
					{Name: "regex", Args: []any{`(\d+)\s*\(\s*(\d+)\s*新\)`}},
					{Name: "index", Args: []any{1}},
					{Name: "parseNumber"},
				},
			},
		},
	},
	Selectors: &v2.SiteSelectors{
		TableRows:       "table.torrents > tbody > tr:has(table.torrentname), table.torrents > tr:has(table.torrentname)",
		Title:           "table.torrentname a[href*='details.php']",
		TitleLink:       "table.torrentname a[href*='details.php']",
		Subtitle:        "table.torrentname td.embedded > span:not(.tags)",
		Size:            "td.rowfollow:nth-child(5)",
		Seeders:         "td.rowfollow:nth-child(6)",
		Leechers:        "td.rowfollow:nth-child(7)",
		Snatched:        "td.rowfollow:nth-child(8)",
		DiscountIcon:    "img[src*='ico_free.gif'], img[src*='ico_free2up.gif'], img[src*='ico_50pctdown.gif']",
		DiscountMapping: map[string]v2.DiscountLevel{
			"ico_free":      v2.DiscountFree,
			"ico_free2up":   v2.Discount2xFree,
			"ico_50pctdown": v2.DiscountPercent50,
		},
		DiscountEndTime: "span.free_end_time[title], font.free span[title], font.twoupfree span[title]",
		Category:        "td.rowfollow:nth-child(1) img[alt]",
		UploadTime:      "td.rowfollow:nth-child(4) span[title]",
	},
	LevelRequirements: []v2.SiteLevelRequirement{
		{
			ID:        0,
			Name:      "Byte",
			Privilege: "新人考核：从注册之日起，30天内上传下载各40G以上，分享率不低于1，做种积分5000。",
		},
		{
			ID:         1,
			Name:       "KiloByte",
			Interval:   "P5W",
			Downloaded: "60GB",
			Ratio:      1.1,
			Privilege:  "此等级为正式会员，可申请种子候选。升级条件：注册满5周，下载量60G以上，分享率大于1.1。小于1.0会自动降级。",
		},
		{
			ID:         2,
			Name:       "MegaByte",
			Interval:   "P8W",
			Downloaded: "150GB",
			Ratio:      2.0,
			Privilege:  "升级条件：注册满8周，下载量150G以上，分享率大于2.0会升级，小于1.9会自动降级。",
		},
		{
			ID:         3,
			Name:       "GigaByte",
			Interval:   "P8W",
			Downloaded: "250GB",
			Ratio:      2.0,
			Privilege:  "此等级可挂起，可进入积分商城。升级条件：注册满8周，下载量250G以上，分享率大于2.0会升级，小于1.9会自动降级。",
		},
		{
			ID:         4,
			Name:       "TeraByte",
			Interval:   "P8W",
			Downloaded: "500GB",
			Ratio:      2.5,
			Privilege:  "此等级可用积分购买邀请，并可浏览全站。升级条件：注册满8周，下载量500G以上，分享率大于2.5会升级，小于2.4会自动降级。",
		},
		{
			ID:         5,
			Name:       "PetaByte",
			Interval:   "P16W",
			Downloaded: "750GB",
			Ratio:      2.5,
			Privilege:  "此等级可直接发布种子。升级条件：注册满16周，下载量750G以上，分享率大于2.5会升级，低于2.4会自动降级。",
		},
		{
			ID:         6,
			Name:       "ExaByte",
			Interval:   "P24W",
			Downloaded: "1TB",
			Ratio:      3.0,
			Privilege:  "此等级自行挂起账号后不会被清除。升级条件：注册满24周，下载量1TB以上，分享率大于3.0会升级，低于2.9自动降级。",
		},
		{
			ID:         7,
			Name:       "ZettaByte",
			Interval:   "P24W",
			Downloaded: "1.5TB",
			Ratio:      3.5,
			Privilege:  "此等级免除流量考核。升级条件：注册满24周，下载量1.5TB以上，分享率大于3.5会升级，低于3.4自动降级。",
		},
		{
			ID:         8,
			Name:       "YottaByte",
			Interval:   "P24W",
			Downloaded: "2.5TB",
			Ratio:      4.0,
			Privilege:  "此等级可查看排行榜。升级条件：注册满24周，下载量2.5TB以上，分享率大于4.0会升级，低于3.9会自动降级。",
		},
		{
			ID:         9,
			Name:       "BrontoByte",
			Interval:   "P32W",
			Downloaded: "3.5TB",
			Ratio:      5.0,
			Privilege:  "此等级及以上用户会永远保留账号。升级条件：注册满32周，下载量3.5TB以上，分享率大于5.0会升级，低于4.9会自动降级。",
		},
		{
			ID:         10,
			Name:       "NonaByte",
			Interval:   "P48W",
			Downloaded: "5TB",
			Uploaded:   "50TB",
			Ratio:      6.0,
			Privilege:  "升级条件：注册满48周，上传量50TB以上，下载量5TB以上，分享率大于6.0会升级，低于5.9会自动降级。",
		},
		{
			ID:         11,
			Name:       "DoggaByte",
			Interval:   "P48W",
			Downloaded: "10TB",
			Uploaded:   "100TB",
			Ratio:      6.0,
			Privilege:  "升级条件：注册满48周，上传量100TB以上，下载量10TB以上，分享率大于6.0会升级，低于5.9会自动降级。",
		},
		{
			ID:        100,
			Name:      "VIP",
			GroupType: v2.LevelGroupVIP,
			Privilege: "为TTG做出特殊重大贡献的用户或合作者等。只计算上传量，不计算下载量。",
		},
	},
	DetailParser: &v2.DetailParserConfig{
		TimeLayout: "2006-01-02 15:04",
		DiscountMapping: map[string]v2.DiscountLevel{
			"ico_free":      v2.DiscountFree,
			"ico_free2up":   v2.Discount2xFree,
			"ico_50pctdown": v2.DiscountPercent50,
		},
		HRKeywords:       []string{}, // TTG不通过常规HR关键词检测
		TitleSelector:    "h1",
		IDSelector:       "a.bookmark",
		DiscountSelector: "img[src*='ico_free']",
		EndTimeSelector:  "font[color='red']",
		SizeSelector:     "td.heading:contains('尺寸')",
		SizeRegex:        `([\d.]+)\s*GB`,
	},
	CreateDriver: createTTGDriver,
}

func init() {
	v2.RegisterSiteDefinition(TTGDefinition)
}

// createTTGDriver 创建 TTG 自定义驱动
func createTTGDriver(config v2.SiteConfig, logger *zap.Logger) (v2.Site, error) {
	var opts v2.NexusPHPOptions
	if len(config.Options) > 0 {
		if err := json.Unmarshal(config.Options, &opts); err != nil {
			return nil, fmt.Errorf("parse TTG options: %w", err)
		}
	}

	if opts.Cookie == "" {
		return nil, fmt.Errorf("TTG 站点需要配置 Cookie")
	}

	siteDef := v2.GetDefinitionRegistry().GetOrDefault(config.ID)

	baseURL := config.BaseURL
	if baseURL == "" && siteDef != nil && len(siteDef.URLs) > 0 {
		baseURL = siteDef.URLs[0]
	}

	// 创建标准 NexusPHP 驱动
	nexusDriver := v2.NewNexusPHPDriver(v2.NexusPHPDriverConfig{
		BaseURL: baseURL,
		Cookie:  opts.Cookie,
	})

	if siteDef != nil {
		nexusDriver.SetSiteDefinition(siteDef)
	}

	// 包装为 ttgDriver
	driver := &ttgDriver{
		NexusPHPDriver: nexusDriver,
	}

	return v2.NewBaseSite(driver, v2.BaseSiteConfig{
		ID:        config.ID,
		Name:      config.Name,
		Kind:      v2.SiteNexusPHP,
		RateLimit: config.RateLimit,
		RateBurst: config.RateBurst,
		Logger:    logger.With(zap.String("site", config.ID)),
	}), nil
}

// GetTorrentDetail 重写以处理 TTG 特殊的免费时间格式
func (d *ttgDriver) GetTorrentDetail(ctx context.Context, guid, link string) (*v2.TorrentItem, error) {
	// 仅使用 Link 字段提取种子 ID，不使用 GUID
	torrentID := extractTTGTorrentIDFromLink(link)
	if torrentID == "" {
		return nil, fmt.Errorf("无法从 link 提取种子 ID: %s", link)
	}

	req, err := d.PrepareDetail(torrentID)
	if err != nil {
		return nil, fmt.Errorf("prepare detail request for torrent %s: %w", torrentID, err)
	}

	res, err := d.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("execute detail request for torrent %s: %w", torrentID, err)
	}

	if res.Document == nil {
		return nil, v2.ErrParseError
	}

	doc := res.Document.Selection

	// 使用标准Parser解析非折扣字段
	parser := v2.NewNexusPHPParserFromDefinition(d.GetSiteDefinition())
	title, torrentID := parser.ParseTitleAndID(doc)
	sizeMB := parser.ParseSizeMB(doc)
	hasHR := parser.ParseHR(doc)

	// 使用TTG自定义方式解析折扣
	discount, discountEnd := parseTTGDiscount(doc)

	siteID := "ttg"
	if def := d.GetSiteDefinition(); def != nil {
		siteID = def.ID
	}

	item := &v2.TorrentItem{
		ID:              torrentID,
		Title:           title,
		SizeBytes:       int64(sizeMB * 1024 * 1024),
		DiscountLevel:   discount,
		DiscountEndTime: discountEnd,
		HasHR:           hasHR,
		SourceSite:      siteID,
	}

	return item, nil
}

// extractTTGTorrentIDFromLink 从 Link URL 中提取种子 ID
func extractTTGTorrentIDFromLink(link string) string {
	if link == "" {
		return ""
	}
	// 从 URL 如 https://totheglory.im/details.php?id=12345 中提取 ID
	parts := strings.Split(link, "id=")
	if len(parts) < 2 {
		return ""
	}
	idPart := parts[1]
	// 处理可能的额外参数，如 &hit=1
	if idx := strings.Index(idPart, "&"); idx != -1 {
		idPart = idPart[:idx]
	}
	return idPart
}

// parseTTGDiscountEndTime 解析 TTG 特殊的免费结束时间格式
// 格式：<font color="red">到期时间为2026-01-30 16:32</font>
func parseTTGDiscountEndTime(doc *goquery.Selection) time.Time {
	// 从红色 font 标签中查找时间
	var endTime time.Time
	timeRe := regexp.MustCompile(`到期时间为(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2})`)

	doc.Find("font[color='red']").Each(func(_ int, s *goquery.Selection) {
		if !endTime.IsZero() {
			return // 已经找到了时间
		}
		text := s.Text()
		matches := timeRe.FindStringSubmatch(text)
		if len(matches) >= 2 {
			// TTG 使用格式 "2026-01-30 16:32" 不包含秒
			if t, err := v2.ParseTimeInCST("2006-01-02 15:04", matches[1]); err == nil {
				endTime = t
			}
		}
	})

	return endTime
}

// parseTTGDiscount 解析TTG特有的免费种子标记
// TTG使用图片src属性而非class属性来标记免费种子
// HTML: <img src="./details_files/ico_free.gif" class="topic">
func parseTTGDiscount(doc *goquery.Selection) (v2.DiscountLevel, time.Time) {
	discountMapping := map[string]v2.DiscountLevel{
		"ico_free":      v2.DiscountFree,
		"ico_free2up":   v2.Discount2xFree,
		"ico_50pctdown": v2.DiscountPercent50,
	}

	var discount v2.DiscountLevel = v2.DiscountNone
	var endTime time.Time

	// 查找所有可能包含折扣标记的图片
	doc.Find("img[src*='ico_free'], img[src*='ico_free2up'], img[src*='ico_50pctdown']").EachWithBreak(func(_ int, el *goquery.Selection) bool {
		src, exists := el.Attr("src")
		if !exists {
			return true
		}

		// 从src中提取文件名
		filename := path.Base(src)
		base := strings.TrimSuffix(filename, path.Ext(filename))

		if level, ok := discountMapping[base]; ok {
			discount = level
			return false
		}
		return true
	})

	// 解析结束时间
	if discount != v2.DiscountNone {
		endTime = parseTTGDiscountEndTime(doc)
	}

	return discount, endTime
}
