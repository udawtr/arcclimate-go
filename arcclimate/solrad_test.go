package arcclimate

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 直散分離のテスト
func Test_SeparateSolarRadiation(t *testing.T) {
	msm_target := MsmTarget{
		date: []time.Time{
			time.Date(2010, time.December, 31, 18, 0, 0, 0, time.UTC),
			time.Date(2010, time.December, 31, 19, 0, 0, 0, time.UTC),
			time.Date(2010, time.December, 31, 20, 0, 0, 0, time.UTC),
		},
		TMP:       []float64{2.044160, 2.340257, 2.554119},
		MR:        []float64{2.439611, 2.525386, 2.659009},
		DSWRF_est: []float64{0.012614, 0.000000, 0.000000},
		DSWRF_msm: nil,
		Ld:        []float64{252.673238, 254.411606, 266.641984},
		VGRD:      []float64{-3.145490, -3.435850, -3.729496},
		UGRD:      []float64{5.996833, 5.851733, 5.702433},
		PRES:      []float64{101056.458196, 101092.748730, 101110.831642},
		APCP01:    []float64{0.109003, 0.077466, 0.083498},
		RH:        []float64{55.897958, 56.673977, 58.781416},
		Pw:        []float64{3.958612, 4.099267, 4.316939},
		DT:        []float64{-5.169731, -4.761409, -4.154089},
	}

	//Nagata
	msm_target.SeparateSolarRadiation(33.8834976, 130.8751773, 2.7, "Nagata")
	assert.True(t, math.Abs(msm_target.h[0]-(-2.695877)) < 0.000001)
	assert.True(t, math.Abs(msm_target.A[0]-243.709298) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[0].SH-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[1].SH-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[2].SH-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[0].DN-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[1].DN-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[2].DN-0.000000) < 0.000001)

	//Watanabe
	msm_target.SeparateSolarRadiation(33.8834976, 130.8751773, 2.7, "Watanabe")
	assert.True(t, math.Abs(msm_target.h[0]-(-2.695877)) < 0.000001)
	assert.True(t, math.Abs(msm_target.A[0]-243.709298) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[0].SH-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[1].SH-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[2].SH-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[0].DN-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[1].DN-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[2].DN-0.000000) < 0.000001)

	//Erbs
	msm_target.SeparateSolarRadiation(33.8834976, 130.8751773, 2.7, "Erbs")
	assert.True(t, math.Abs(msm_target.h[0]-(-2.695877)) < 0.000001)
	assert.True(t, math.Abs(msm_target.A[0]-243.709298) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[0].SH-0.012674) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[1].SH-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[2].SH-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[0].DN-0.001273) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[1].DN-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[2].DN-0.000000) < 0.000001)

	//Udagawa
	msm_target.SeparateSolarRadiation(33.8834976, 130.8751773, 2.7, "Udagawa")
	assert.True(t, math.Abs(msm_target.h[0]-(-2.695877)) < 0.000001)
	assert.True(t, math.Abs(msm_target.A[0]-243.709298) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[0].DN-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[1].DN-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[2].DN-0.000000) < 0.000001)
	assert.True(t, math.Abs(msm_target.SR_est[0].SH-0.012614) < 1.0e-6)
	assert.True(t, math.Abs(msm_target.SR_est[1].SH-0.000000) < 1.0e-6)
	assert.True(t, math.Abs(msm_target.SR_est[2].SH-0.000000) < 1.0e-6)

	//Perez
	msm_target.SeparateSolarRadiation(33.8834976, 130.8751773, 2.7, "Perez")
	assert.True(t, math.Abs(msm_target.h[0]-(-2.695877)) < 1.0e-6)
	assert.True(t, math.Abs(msm_target.A[0]-243.709298) < 1.0e-6)
	assert.True(t, math.Abs(msm_target.SR_est[0].DN-0.000000) < 1.0e-6)
	assert.True(t, math.Abs(msm_target.SR_est[1].DN-0.000000) < 1.0e-6)
	assert.True(t, math.Abs(msm_target.SR_est[2].DN-0.000000) < 1.0e-6)
	assert.True(t, math.Abs(msm_target.SR_est[0].SH-0.012614) < 1.0e-6)
	assert.True(t, math.Abs(msm_target.SR_est[1].SH-0.000000) < 1.0e-6)
	assert.True(t, math.Abs(msm_target.SR_est[2].SH-0.000000) < 1.0e-6)
}

// Erbsモデルのテスト
func Test_get_SH_Erbs(t *testing.T) {
	// 晴天指数 KT ごとに処理が異なる

	// KT = 0.44 / (1*1) <= 0.22
	SH1 := get_SH_Erbs([]float64{0.44}, []float64{1}, []float64{1})
	assert.True(t, math.Abs(SH1[0]-0.34105039) < 1.0e-8)

	// KT = 1.60 / (1*1) <= 0.80
	SH2 := get_SH_Erbs([]float64{1.60}, []float64{1}, []float64{1})
	assert.True(t, math.Abs(SH2[0]-0.264) < 1.0e-8)

	// KT = 1.61 / (1*1) > 0.80
	SH3 := get_SH_Erbs([]float64{1.61}, []float64{1}, []float64{1})
	assert.True(t, math.Abs(SH3[0]-0.26565) < 1.0e-8)
}

// Udagawa モデルのテスト
func Test_get_DN_Udagawa(t *testing.T) {
	// KC = 1.027... => 1次式
	DT1 := get_DN_Udagawa([]float64{0.2}, []float64{3}, []float64{0.5})
	assert.True(t, math.Abs(DT1[0]-0.01214507) < 1.0e-8)

	// KC = 0.6848075, KT = 0.2 => 3次式
	DT2 := get_DN_Udagawa([]float64{0.2}, []float64{2}, []float64{0.5})
	assert.True(t, math.Abs(DT2[0]-0.0273264) < 1.0e-8)
}

func Test_IndexOf_RealValue(t *testing.T) {
	K := IndexOf(0.02543524509559203, []float64{0.015, 0.035, 0.07, 0.15, 0.3})
	assert.Equal(t, K, 1)
}

func Test_IndexOf_NaN(t *testing.T) {
	K := IndexOf(math.NaN(), []float64{0.015, 0.035, 0.07, 0.15, 0.3})
	assert.Equal(t, K, 5)
}
