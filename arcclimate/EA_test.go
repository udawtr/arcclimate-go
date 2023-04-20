package arcclimate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_get_smoothing_months(t *testing.T) {
	// すべて2000年: 閏年のため3月はスムージング対象
	assert.Equal(t,
		SmoothingMonths([]int{2000, 2000, 2000, 2000, 2000, 2000, 2000, 2000, 2000, 2000, 2000, 2000}),
		[]SmootingMonth{{3, 2000, 2000}},
	)

	// すべて2001年: 閏年ではないためスムージング対象なし
	assert.Equal(t,
		SmoothingMonths([]int{2001, 2001, 2001, 2001, 2001, 2001, 2001, 2001, 2001, 2001, 2001, 2001}),
		[]SmootingMonth{},
	)

	// 毎月代表年が違う場合
	assert.Equal(t,
		SmoothingMonths([]int{2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011}),
		[]SmootingMonth{
			{1, 2011, 2000},
			{2, 2000, 2001},
			{3, 2001, 2002},
			{4, 2002, 2003},
			{5, 2003, 2004},
			{6, 2004, 2005},
			{7, 2005, 2006},
			{8, 2006, 2007},
			{9, 2007, 2008},
			{10, 2008, 2009},
			{11, 2009, 2010},
			{12, 2010, 2011},
		},
	)
}
