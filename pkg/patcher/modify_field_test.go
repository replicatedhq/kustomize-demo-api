package patcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModifyField(t *testing.T) {
	tests := []struct {
		name     string
		original []byte
		path     []string
		want     []byte
		wantErr  bool
	}{
		{
			name: "path in service",
			original: []byte(`
apiVersion: v1
kind: Service
metadata:
  name: redis-master
  labels:
    app: redis
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: redis
    role: master
    tier: "backend"
`),
			path: []string{"spec", "selector", "app"},
			want: []byte(`apiVersion: v1
kind: Service
metadata:
  labels:
    app: redis
  name: redis-master
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: TO_BE_MODIFIED
    role: master
    tier: backend
`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := ModifyField(tt.original, tt.path)
			req.NoError(err)
			req.Equal(string(tt.want), string(got))
		})
	}
}
