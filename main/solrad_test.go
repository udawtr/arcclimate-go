package main // mainパッケージであることを宣言

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IndexOf_RealValue(t *testing.T) {
	K := IndexOf(0.02543524509559203, []float64{0.015, 0.035, 0.07, 0.15, 0.3})
	assert.Equal(t, K, 1)
}

func Test_IndexOf_NaN(t *testing.T) {
	K := IndexOf(math.NaN(), []float64{0.015, 0.035, 0.07, 0.15, 0.3})
	assert.Equal(t, K, 5)
}
