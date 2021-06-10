package wrkpool

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReadUrlResultIsNotNil(t *testing.T) {
	urlR := NewUrlReader("http://example.com/")
	res, err := urlR.Read()
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestReadUrlResultIsNil(t *testing.T) {
	urlR := NewUrlReader()
	_, err := urlR.Read()
	require.Error(t, err)
}


func TestReadMultiplyUrlResultIsNotNil(t *testing.T) {
	urlR := NewUrlReader("http://example.com/", "https://www.instagram.com/", "http://example.com/")
	res, err := urlR.Read()
	require.NoError(t, err)
	require.NotNil(t, res)

	require.True(t, len(res) == 3)
}

func TestReadWithError(t *testing.T) {
	urlR := NewUrlReader("http://example.com/", "", "http://example.com/")
	res, err := urlR.Read()
	require.NoError(t, err)
	require.NotNil(t, res)

	require.True(t, len(res) == 3)

	for i := 0; i < 3; i++ {
		if i == 1 {
			require.Error(t, res[i].err)
		} else {
			require.NoError(t, res[i].err)
		}
	}
}

