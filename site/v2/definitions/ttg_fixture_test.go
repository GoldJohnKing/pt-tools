package definitions

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2 "github.com/sunerpy/pt-tools/site/v2"
)

func TestParseTTGDiscount(t *testing.T) {
	// 读取样例HTML
	html, err := os.ReadFile("../../../sample-ttg/details.php")
	require.NoError(t, err)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	require.NoError(t, err)

	discount, endTime := parseTTGDiscount(doc.Selection)

	assert.Equal(t, v2.DiscountFree, discount)
	assert.False(t, endTime.IsZero())
	// 验证结束时间是2026-02-08 22:30
	expectedTime := time.Date(2026, 2, 8, 22, 30, 0, 0, v2.CSTLocation)
	assert.Equal(t, expectedTime, endTime)
}
