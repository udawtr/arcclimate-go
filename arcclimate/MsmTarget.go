package arcclimate

import (
	"sort"
	"time"
)

// 推定結果データ
type MsmTarget struct {
	date []time.Time //1.参照時刻。日本標準時JST

	//標高補正のみのデータ項目
	TMP       []float64 //2.参照時刻時点の気温の瞬時値 (単位:℃)
	MR        []float64 //3.参照時刻時点の重量絶対湿度の瞬時値 (単位:g/kgDA)
	DSWRF_est []float64 //4.参照時刻の前1時間の推定日射量の積算値 (単位:MJ/m2)
	DSWRF_msm []float64 //5.参照時刻の前1時間の日射量の積算値 (単位:MJ/m2)
	Ld        []float64 //6.大気放射量 W/m2 or MJ/m2
	VGRD      []float64 //7.南北風(V軸) (単位:m/s)
	UGRD      []float64 //8.東西風(U軸) (単位:m/s)
	PRES      []float64 //9.気圧 (単位:hPa)
	APCP01    []float64 //10.参照時刻の前1時間の降水量の積算値 (単位:mm/h)

	DSWRF []float64 //標準年の計算時に使用するDSWRF

	//追加項目
	w_spd []float64 //11.参照時刻時点の風速の瞬時値 (単位:m/s)
	w_dir []float64 //12.参照時刻時点の風向の瞬時値 (単位:°)
	h     []float64 //13.参照時刻時点の太陽高度角 (単位:°)
	A     []float64 //14.参照時刻時点の太陽方位角 (単位:°)

	NR []float64 //夜間放射量[MJ/m2]

	RH []float64 //	float64: 相対湿度[%]
	Pw []float64 //	float64: 水蒸気分圧 [hpa]

	//計算対象時刻の露点温度(℃)
	DT []float64

	//直散分離用
	SR_est []SolarRadiation //直散分離結果(推定日射量 DSWRF_est に基づく)
	SR_msm []SolarRadiation //直散分離結果(日射量 DSWRF_msm に基づく)
}

// 開始年 start_year から 終了年 end_year までのデータを抜き出して新しい構造体を作成します。
func (df_msm *MsmTarget) ExctactMsmYear(start_year int, end_year int) *MsmTarget {
	start_time := time.Date(start_year, 1, 1, 0, 0, 0, 0, time.UTC)
	end_time := time.Date(end_year, 12, 31, 23, 0, 0, 0, time.UTC)
	return df_msm.ExctactMsm(start_time, end_time)
}

// 開始日時 start_time から 終了日時 end_time までのデータを抜き出して新しい構造体を作成します。
func (df_msm *MsmTarget) ExctactMsm(start_time time.Time, end_time time.Time) *MsmTarget {
	start_index := sort.Search(len(df_msm.date), func(i int) bool {
		return df_msm.date[i].After(start_time) || df_msm.date[i].Equal(start_time)
	})
	end_index := sort.Search(len(df_msm.date), func(i int) bool {
		return df_msm.date[i].After(end_time) || df_msm.date[i].Equal(end_time)
	})
	msm := MsmTarget{
		date:   append([]time.Time{}, df_msm.date[start_index:end_index+1]...),
		TMP:    append([]float64{}, df_msm.TMP[start_index:end_index+1]...),
		MR:     append([]float64{}, df_msm.MR[start_index:end_index+1]...),
		Ld:     append([]float64{}, df_msm.Ld[start_index:end_index+1]...),
		VGRD:   append([]float64{}, df_msm.VGRD[start_index:end_index+1]...),
		UGRD:   append([]float64{}, df_msm.UGRD[start_index:end_index+1]...),
		PRES:   append([]float64{}, df_msm.PRES[start_index:end_index+1]...),
		APCP01: append([]float64{}, df_msm.APCP01[start_index:end_index+1]...),
		h:      append([]float64{}, df_msm.h[start_index:end_index+1]...),
		A:      append([]float64{}, df_msm.A[start_index:end_index+1]...),
		RH:     append([]float64{}, df_msm.RH[start_index:end_index+1]...),
		Pw:     append([]float64{}, df_msm.Pw[start_index:end_index+1]...),
		NR:     append([]float64{}, df_msm.NR[start_index:end_index+1]...),
		DT:     append([]float64{}, df_msm.DT[start_index:end_index+1]...),
		SR_est: append([]SolarRadiation{}, df_msm.SR_est[start_index:end_index+1]...),
		SR_msm: append([]SolarRadiation{}, df_msm.SR_msm[start_index:end_index+1]...),
	}
	if df_msm.DSWRF != nil {
		msm.DSWRF = append([]float64{}, df_msm.DSWRF[start_index:end_index+1]...)
	}
	if df_msm.DSWRF_est != nil {
		msm.DSWRF_est = append([]float64{}, df_msm.DSWRF_est[start_index:end_index+1]...)
	}
	if df_msm.DSWRF_msm != nil {
		msm.DSWRF_msm = append([]float64{}, df_msm.DSWRF_msm[start_index:end_index+1]...)
	}
	if df_msm.w_spd != nil {
		msm.w_spd = append([]float64{}, df_msm.w_spd[start_index:end_index+1]...)
	}
	if df_msm.w_dir != nil {
		msm.w_dir = append([]float64{}, df_msm.w_dir[start_index:end_index+1]...)
	}

	return &msm
}

// 2月29日を除外して新しい構造体を作成します。
func (df_msm *MsmTarget) filterMsmLeapYear29th() *MsmTarget {
	date := []time.Time{}
	TMP := []float64{}
	MR := []float64{}
	DSWRF := []float64{}
	Ld := []float64{}
	VGRD := []float64{}
	UGRD := []float64{}
	PRES := []float64{}
	APCP01 := []float64{}
	h := []float64{}
	A := []float64{}
	RH := []float64{}
	Pw := []float64{}
	NR := []float64{}
	DT := []float64{}
	AAA_est := []SolarRadiation{}
	AAA_msm := []SolarRadiation{}
	// w_spd := []float64{}
	// w_dir := []float64{}

	for i := 0; i < len(df_msm.date); i++ {
		if !(df_msm.date[i].Month() == 2 && df_msm.date[i].Day() == 29) {
			date = append(date, df_msm.date[i])
			TMP = append(TMP, df_msm.TMP[i])
			MR = append(MR, df_msm.MR[i])
			DSWRF = append(DSWRF, df_msm.DSWRF[i])
			Ld = append(Ld, df_msm.Ld[i])
			VGRD = append(VGRD, df_msm.VGRD[i])
			UGRD = append(UGRD, df_msm.UGRD[i])
			PRES = append(PRES, df_msm.PRES[i])
			APCP01 = append(APCP01, df_msm.APCP01[i])
			h = append(h, df_msm.h[i])
			A = append(A, df_msm.A[i])
			RH = append(RH, df_msm.RH[i])
			Pw = append(Pw, df_msm.Pw[i])
			NR = append(NR, df_msm.NR[i])
			DT = append(DT, df_msm.DT[i])
			AAA_est = append(AAA_est, df_msm.SR_est[i])
			AAA_msm = append(AAA_msm, df_msm.SR_msm[i])
			// w_spd = append(w_spd, df_msm.w_dir[i])
			// w_dir = append(w_dir, df_msm.w_dir[i])
		}
	}

	return &MsmTarget{
		date:   date,
		TMP:    TMP,
		MR:     MR,
		DSWRF:  DSWRF,
		Ld:     Ld,
		VGRD:   VGRD,
		UGRD:   UGRD,
		PRES:   PRES,
		APCP01: APCP01,
		h:      h,
		A:      A,
		RH:     RH,
		Pw:     Pw,
		NR:     NR,
		DT:     DT,
		SR_est: AAA_est,
		SR_msm: AAA_msm,
		// w_spd:     w_spd,
		// w_dir:     w_dir,
	}
}
