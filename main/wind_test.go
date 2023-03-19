package main // mainパッケージであることを宣言

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_wind16(t *testing.T) {
	spd, dir := get_wind16(1.0, 1.0)
	assert.InDelta(t, 1.4141456, spd, 0.0001)
	assert.Equal(t, 180.0+45.0, dir)
}
