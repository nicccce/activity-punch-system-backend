package test

import (
	"activity-punch-system-backend/internal/global/response"
	"github.com/stretchr/testify/require"
	"testing"
)

func ErrorEqual(t *testing.T, expected *response.Error, resp response.ResponseBody) {
	require.Equal(t, expected.Code, resp.Code)
	require.Equal(t, expected.Message, resp.Msg)
}

func NoError(t *testing.T, resp response.ResponseBody) {
	require.Equal(t, int32(200), resp.Code)
}
