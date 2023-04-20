package arcclimate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 2点間の距離の計算のテスト(GRS80)
// Notes:
//
//	期待値は国土地理院の計算プログラムから取得しました。
//	https://vldb.gsi.go.jp/sokuchi/surveycalc/surveycalc/bl2stf.html
func Test_vincentyInverse(t *testing.T) {
	lat1 := 36.10377477777778
	lon1 := 140.08785502777778
	lat2 := 35.65502847222223
	lon2 := 139.74475044444443

	L := vincentyInverse(lat1, lon1, lat2, lon2)
	assert.InDelta(t, L, 58643.804, 0.01)
}

// 2点間の距離の計算のテスト(同じ拠点)
func Test_vincentyInverse_SamePosition(t *testing.T) {
	lat1 := 36.10377477777778
	lon1 := 140.08785502777778
	lat2 := 36.10377477777778
	lon2 := 140.08785502777778

	L := vincentyInverse(lat1, lon1, lat2, lon2)
	assert.Equal(t, L, 0.0)
}
