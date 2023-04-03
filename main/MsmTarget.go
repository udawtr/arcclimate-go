package main

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

	NR []float64 //夜間放射量[MJ/m2]

	RH []float64 //	float64: 相対湿度[%]
	Pw []float64 //	float64: 水蒸気分圧 [hpa]

	//計算対象時刻の露点温度(℃)
	DT []float64

	//直散分離
	AAA_est []AAA
	AAA_msm []AAA
}

func (df_msm *MsmTarget) ExctactMsm(start_time time.Time, end_time time.Time) MsmTarget {
	start_index := sort.Search(len(df_msm.date), func(i int) bool {
		return df_msm.date[i].After(start_time) || df_msm.date[i].Equal(start_time)
	})
	end_index := sort.Search(len(df_msm.date), func(i int) bool {
		return df_msm.date[i].After(end_time) || df_msm.date[i].Equal(end_time)
	})
	msm := MsmTarget{
		date:    append([]time.Time{}, df_msm.date[start_index:end_index+1]...),
		TMP:     append([]float64{}, df_msm.TMP[start_index:end_index+1]...),
		MR:      append([]float64{}, df_msm.MR[start_index:end_index+1]...),
		DSWRF:   append([]float64{}, df_msm.DSWRF[start_index:end_index+1]...),
		Ld:      append([]float64{}, df_msm.Ld[start_index:end_index+1]...),
		VGRD:    append([]float64{}, df_msm.VGRD[start_index:end_index+1]...),
		UGRD:    append([]float64{}, df_msm.UGRD[start_index:end_index+1]...),
		PRES:    append([]float64{}, df_msm.PRES[start_index:end_index+1]...),
		APCP01:  append([]float64{}, df_msm.APCP01[start_index:end_index+1]...),
		RH:      append([]float64{}, df_msm.RH[start_index:end_index+1]...),
		Pw:      append([]float64{}, df_msm.Pw[start_index:end_index+1]...),
		NR:      append([]float64{}, df_msm.NR[start_index:end_index+1]...),
		DT:      append([]float64{}, df_msm.DT[start_index:end_index+1]...),
		AAA_est: append([]AAA{}, df_msm.AAA_est[start_index:end_index+1]...),
		AAA_msm: append([]AAA{}, df_msm.AAA_msm[start_index:end_index+1]...),
	}
	if df_msm.w_spd != nil {
		msm.w_spd = append([]float64{}, df_msm.w_spd[start_index:end_index+1]...)
	}
	if df_msm.w_dir != nil {
		msm.w_dir = append([]float64{}, df_msm.w_dir[start_index:end_index+1]...)
	}

	return msm
}
