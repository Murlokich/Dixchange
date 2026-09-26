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
