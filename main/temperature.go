package main

//気温の関するモジュール

// 気温の標高補正を行います。
// TMP: 気温 [℃]
// ele_gap: 標高差 [m]
// 標高補正後の気温 [C]を返します。
// ただし、気温減率の平均値を0.0065℃/mとする。
func get_corrected_TMP(TMP float64, ele_gap float64) float64 {
	return TMP + ele_gap*-0.0065
}
