package http //nolint:testpackage // tests binding of the unexported request handling

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetOrdersRequest_BindQuery(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		query        string
		wantStatuses []string
		wantErr      bool
	}{
		{name: "single", query: "status=open", wantStatuses: []string{"open"}},
		{
			name:         "comma separated",
			query:        "status=open,partially_filled",
			wantStatuses: []string{"open", "partially_filled"},
		},
		{name: "missing", query: "", wantErr: true},
		{name: "empty", query: "status=", wantErr: true},
		{name: "trailing comma", query: "status=open,", wantErr: true},
		{name: "unsupported status", query: "status=open,filled", wantErr: true},
		{name: "wrong case", query: "status=Open", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/orders?"+tc.query, nil)

			var req GetOrdersRequest
			err := c.ShouldBindQuery(&req)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantStatuses, req.Statuses)
		})
	}
}

func TestGetTradesRequest_BindQuery(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		query      string
		wantCursor string
		wantErr    bool
	}{
		{name: "first page", query: "", wantCursor: ""},
		{
			name:       "cursor",
			query:      "cursor=019f7f2c-0caa-70ab-b1ec-414f744c3e33",
			wantCursor: "019f7f2c-0caa-70ab-b1ec-414f744c3e33",
		},
		{name: "not a uuid", query: "cursor=2026-09-28", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/trades?"+tc.query, nil)

			var req GetTradesRequest
			err := c.ShouldBindQuery(&req)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantCursor, req.Cursor)
		})
	}
}
