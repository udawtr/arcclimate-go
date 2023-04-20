package arcclimate

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CorrectMR(t *testing.T) {
	// 温度 [C]
	TMP := 20.0

	// 気圧 [hPa]
	PRES := 1013.25

	// 重量絶対湿度 [g/kg(DA)]
	MR_sat := mixingRatio(PRES, TMP)
	assert.InDelta(t, 1437.53, MR_sat, 0.01)

	// 重量絶対湿度の標高補正1
	// 1437.53 < 8300 なので、 補正結果は 1437.53
	MR_corr := CorrectMR(8300.0, TMP, PRES)
	assert.InDelta(t, 1437.53, MR_corr, 0.01)

	// 重量絶対湿度の標高補正2
	// 1437 .53 > 300.0 なので、 補正結果は 300.0
	MR_corr = CorrectMR(300.0, TMP, PRES)
	assert.InDelta(t, 300.0, MR_corr, 0.01)
}

func Test_mixingRatio(t *testing.T) {
	// 気圧 [hPa]
	PRES := 1013.25

	// 絶対温度 [K]
	T := 293.15

	// 乾燥空気の気体定数 [J/kgK]
	Rd := 287.0

	// 飽和水蒸気圧 [hPa]
	eSAT := eSAT(T)

	// 飽和水蒸気量 [g/m3]
	aT := aT(eSAT, T)

	// 重量絶対湿度 [g/kg(DA)]
	MR := aT / (PRES / (Rd * T))

	assert.InDelta(t, MR, mixingRatio(PRES, 20.0), 0.00001)
}

func Test_eSAT(t *testing.T) {
	// 絶対温度 [K]
	T := 293.15

	ln_P := -5800.2206/T + 1.3914993 - 0.048640239*T +
		0.000041764768*T*T -
		0.000000014452093*T*T*T +
		6.5459673*math.Log(T)

	// 飽和水蒸気圧 [Pa]
	P := math.Exp(ln_P)

	assert.InDelta(t, P/100, eSAT(T), 0.00001)
}

func Test_aT(t *testing.T) {
	// 飽和水蒸気圧 [hPa]
	eSAT := 40.1

	// 絶対温度 [K]
	T := 293.15

	// 飽和水蒸気量 [g/m3]
	_aT := 217 * 40.1 / 293.15

	assert.InDelta(t, _aT, aT(eSAT, T), 0.00000000001)
}

func Test_VH(t *testing.T) {
	// 飽和水蒸気量 [g/m3]
	aT := 100.0

	// 相対湿度 [%]
	RH := 60.0

	// 容積絶対湿度 [g/m3]
	_VH := 100 * 0.6

	assert.InDelta(t, _VH, VH(aT, RH), 0.0000000001)
}
