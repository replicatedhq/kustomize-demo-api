package patcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyPatch(t *testing.T) {
	tests := []struct {
		name     string
		original []byte
		patch    []byte
		want     []byte
		wantErr  bool
	}{
		{
			name: "basic deployment",
			original: []byte(`
apiVersion: v1
kind: Service
metadata:
  name: redis-master
  labels:
    app: redis
    role: master
    tier: backend
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: redis
    role: master
    tier: "backend"
`),
			patch: []byte(`
apiVersion: v1
kind: Service
metadata:
  labels:
    app: redis
    role: master
    tier: backend
  name: redis-master
spec:
  selector:
    zquotenum: "456"
`),
			want: []byte(`apiVersion: v1
kind: Service
metadata:
  labels:
    app: redis
    role: master
    tier: backend
  name: redis-master
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: redis
    role: master
    tier: backend
    zquotenum: "456"
`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := ApplyPatch(tt.original, tt.patch)
			req.NoError(err)
			req.Equal(string(tt.want), string(got))
		})
	}
}
