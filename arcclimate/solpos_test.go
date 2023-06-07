package arcclimate

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 太陽位置の計算のテスト
// Python実装との値の一致を確認
func Test_get_sun_position(t *testing.T) {

	solpos := get_sun_position(33.8834976, 130.8751773, []time.Time{
		time.Date(2010, time.December, 31, 18, 0, 0, 0, time.UTC),
	})

	assert.Equal(t, len(solpos), 1)

	assert.True(t, math.Abs(solpos[0].h-(-2.695877)) < 1.0e-6)
	assert.True(t, math.Abs(solpos[0].A-243.709298) < 1.0e-6)
	assert.True(t, math.Abs(solpos[0].IN0-5.083008) < 1.0e-6)
	assert.True(t, math.Abs(solpos[0].Sinh-(-0.047035)) < 1.0e-6)
}
