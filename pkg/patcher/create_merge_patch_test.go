package patcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateTwoWayMergePatch(t *testing.T) {
	tests := []struct {
		name     string
		original []byte
		modified []byte
		want     []byte
	}{
		{
			name: "basic service",
			original: []byte(`
apiVersion: v1
kind: Service
metadata:
  name: redis-master
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: redis
    role: master
    tier: "backend"
`),
			modified: []byte(`apiVersion: v1
kind: Service
metadata:
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
			want: []byte(`apiVersion: v1
kind: Service
metadata:
  name: redis-master
spec:
  selector:
    zquotenum: "456"
`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := CreateTwoWayMergePatch(tt.original, tt.modified)
			req.NoError(err)
			req.Equal(string(tt.want), string(got))
		})
	}
}

func TestCombinePatches(t *testing.T) {
	tests := []struct {
		name     string
		original []byte
		patches  [][]byte
		want     []byte
	}{

		{
			name: "basic service",
			original: []byte(`
apiVersion: v1
kind: Service
metadata:
  name: redis-master
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: redis
    role: master
    tier: "backend"
`),
			patches: [][]byte{
				[]byte(`apiVersion: v1
kind: Service
metadata:
  name: redis-master
spec:
  selector:
    zquotenum: "456"
`),
				[]byte(`apiVersion: v1
kind: Service
metadata:
  name: redis-master
spec:
  selector:
    anotherpatch: "another"
`),
			},
			want: []byte(`apiVersion: v1
kind: Service
metadata:
  name: redis-master
spec:
  selector:
    anotherpatch: another
    zquotenum: "456"
`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := CombinePatches(tt.original, tt.patches)
			req.NoError(err)
			req.Equal(string(tt.want), string(got))
		})
	}
}
