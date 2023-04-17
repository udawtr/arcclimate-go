package main

import (
	"math"
	"time"
)

// MSMファイルから読み取ったデータ
type MsmData struct {
	name string //ファイル名
	Rows *[87687]MsmDataRow
}

type MsmDataRow struct {
	date      time.Time `csv:"date"`                //参照時刻。日本標準時JST
	TMP       float64   `csv:"TMP"`                 //参照時刻時点の気温の瞬時値 (単位:℃)
	MR        float64   `csv:"MR"`                  //参照時刻時点の重量絶対湿度の瞬時値 (単位:g/kgDA)
	DSWRF_est float64   `csv:"DSWRF_est"`           //参照時刻の前1時間の推定日射量の積算値 (単位:MJ/m2)
	DSWRF_msm float64   `csv:"DSWRF_msm,omitempty"` //参照時刻の前1時間の日射量の積算値 (単位:MJ/m2)
	Ld        float64   `csv:"Ld"`                  //参照時刻の前1時間の下向き大気放射量の積算値 (単位:MJ/m2)
	VGRD      float64   `csv:"VGRD"`                //南北風(V軸) (単位:m/s)
	UGRD      float64   `csv:"UGRD"`                //東西風(U軸) (単位:m/s)
	PRES      float64   `csv:"PRES"`                //気圧 (単位:hPa)
	APCP01    float64   `csv:"APCP01"`              //参照時刻の前1時間の降水量の積算値 (単位:mm/h)
}

type MsmDataSet struct {
	Data [4]MsmData
}

func (msm *MsmData) Length() int {
	return len(msm.Rows)
}

// MSMデータフレームの気温 TMP 、気圧 PRES、重量絶対湿度 MR を標高補正する(標高 elevation [m] から ele_target [m] へ補正)。
func (msm *MsmData) CorrectedMsm_TMP_PRES_MR(elevation float64, ele_target float64) *MsmData {

	// 標高差
	ele_gap := ele_target - elevation

	for i := 0; i < msm.Length(); i++ {

		TMP := msm.Rows[i].TMP
		PRES := msm.Rows[i].PRES
		MR := msm.Rows[i].MR

		// 気温補正
		TMP_corr := CorrectTMP(TMP, ele_gap)

		// 気圧補正
		PRES_corr := CorrectPRES(PRES, ele_gap, TMP_corr)

		// 重量絶対湿度補正
		MR_corr := CorrectMR(MR, TMP_corr, PRES_corr)

		// 補正値をデータフレームに戻す
		msm.Rows[i].TMP = TMP_corr
		msm.Rows[i].PRES = PRES_corr
		msm.Rows[i].MR = MR_corr
	}

	// なぜ 気圧消すのか？
	// msm.drop(['PRES'], axis=1, inplace=True)

	return msm
}

//--------------------------------------
// 温度補正
//--------------------------------------

// 気温 TMP [℃] を、 標高差 ele_gap [m] を用いて補正します。
// ただし、気温減率の平均値を0.0065℃/m とします。
func CorrectTMP(TMP float64, ele_gap float64) float64 {
	return TMP + ele_gap*-0.0065
}

//--------------------------------------
// 気圧
//--------------------------------------

// 気圧 PRES [hPa] を 標高差 ele_gap [m] と 気温 TMP [℃]を用いて補正します。
// ただし、気温減率の平均値を0.0065℃/mとします。
func CorrectPRES(PRES float64, ele_gap float64, TMP float64) float64 {
	return PRES * math.Pow(1-((ele_gap*0.0065)/(TMP+273.15)), 5.257)
}

//--------------------------------------
// 重量絶対湿度の計算
//--------------------------------------

// 重量絶対湿度 MR [g/kg(DA)] を 気温 TMP [℃] と 気圧 PRES [hPa] を用いて補正します。
func CorrectMR(MR float64, TMP float64, PRES float64) float64 {
	//  飽和水蒸気量（重量絶対湿度） [g/kg(DA)]
	MR_sat := mixingRatio(PRES, TMP)

	// 重量絶対湿度の補正
	MR_corr := math.Min(MR, MR_sat) // 飽和水蒸気量（重量絶対湿度）を最大とする

	return MR_corr
}

// 気圧 PRES [hPa] と 気温 TMP [℃] から 重量絶対湿度 [g/kg(DA)] を求める。
func mixingRatio(PRES float64, TMP float64) float64 {
	// 絶対温度 [K]
	T := TMP + 273.15

	// 飽和水蒸気圧 [hPa]
	eSAT := eSAT(T)

	// 飽和水蒸気量 [g/m3]
	aT := aT(eSAT, T)

	// 重量絶対湿度 [g/kg(DA)]
	MR := aT / ((PRES / 100) / (2.87 * T))

	return MR
}

// 絶対温度 T [K] から 飽和水蒸気圧 [hPa] を求める。
func eSAT(T float64) float64 {
	return math.Exp(-5800.2206/T+
		1.3914993-0.048640239*T+
		0.41764768*0.0001*T*T-
		0.14452093*0.0000001*T*T*T+
		6.5459673*math.Log(T)) / 100
}

// 飽和水蒸気圧 eSAT [hPa] と 絶対温度 T [K] から 飽和水蒸気量 aT [g/m^3] を求める。
func aT(eSAT float64, T float64) float64 {
	return (217 * eSAT) / T
}

// 飽和水蒸気量 aT [g/m^3] と 相対湿度 RH [%] から 容積絶対湿度 [g/m^3] を求める。
func VH(aT float64, RH float64) float64 {
	return aT * (RH / 100)
}
