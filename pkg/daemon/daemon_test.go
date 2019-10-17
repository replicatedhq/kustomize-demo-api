package daemon

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDaemon(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		postContents string
		expectResult string
	}{
		{
			name:     "add to existing patch",
			endpoint: "/kustomize/patch",
			postContents: `
{
    "path": ["rules",0,"resources",1],
    "original": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - namespaces\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n",
    "existing_patch": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - hasBeenModified\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n"
}`,
			expectResult: `{
    "patch": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - hasBeenModified\n  - TO_BE_MODIFIED\n  verbs:\n  - get\n  - list\n  - watch\n"
}`,
		},
		{
			name:     "add to existing unrelated patch",
			endpoint: "/kustomize/patch",
			postContents: `
{
    "path": ["rules",0,"resources",1],
    "original": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - namespaces\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n",
    "existing_patch": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - namespaces\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n  - hasBeenAdded\n"
}`,
			expectResult: `{
    "patch": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - namespaces\n  - TO_BE_MODIFIED\n  verbs:\n  - get\n  - list\n  - watch\n  - hasBeenAdded\n"
}`,
		},
		{
			name:     "add to existing TO_BE_MODIFIED patch",
			endpoint: "/kustomize/patch",
			postContents: `
{
    "path": ["rules",0,"resources",1],
    "original": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - namespaces\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n",
    "existing_patch": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - TO_BE_MODIFIED\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n"
}`,
			expectResult: `{
    "patch": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - TO_BE_MODIFIED\n  - TO_BE_MODIFIED\n  verbs:\n  - get\n  - list\n  - watch\n"
}`,
		},
		{
			name:     "create new patch",
			endpoint: "/kustomize/patch",
			postContents: `
{
    "path": ["metadata","labels","app"],
    "original": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - namespaces\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n",
    "existing_patch": ""
}`,
			expectResult: `{
    "patch": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\n"
}`,
		},
		{
			name:     "apply patch",
			endpoint: "/kustomize/apply",
			postContents: `
{
    "resource": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - namespaces\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n",
    "patch": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeatMod\n    release: auditbeat\n  name: auditbeat\n"
}`,
			expectResult: `{
    "modified": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeatMod\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - namespaces\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n"
}`,
		},
		{
			name:     "apply nonexistent patch",
			endpoint: "/kustomize/apply",
			postContents: `
{
    "resource": "apiVersion: rbac.authorization.k8s.io/v1\n\n\n\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - namespaces\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n",
    "patch": ""
}`,
			expectResult: `{
    "modified": "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  labels:\n    app: auditbeat\n    release: auditbeat\n  name: auditbeat\nrules:\n- apiGroups:\n  - \"\"\n  resources:\n  - namespaces\n  - pods\n  verbs:\n  - get\n  - list\n  - watch\n"
}`,
		},
		{
			name:     "generate general kustomization",
			endpoint: "/kustomize/generate",
			postContents: `
{
    "resources": ["mypath.yaml", "mypath2.yaml"],
    "patches": ["mypatchpath.yaml"],
    "bases": ["../../mybase"]
}`,
			expectResult: `{
    "kustomization": "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nbases:\n- ../../mybase\nresources:\n- mypath.yaml\n- mypath2.yaml\npatchesStrategicMerge:\n- mypatchpath.yaml\n"
}`,
		},
		{
			name:     "generate base kustomization",
			endpoint: "/kustomize/generate-base",
			postContents: `
{
    "resources": ["mypath.yaml", "mypath2.yaml"]
}`,
			expectResult: `{
    "kustomization": "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- mypath.yaml\n- mypath2.yaml\n"
}`,
		},
		{
			name:     "generate overlay kustomization",
			endpoint: "/kustomize/generate-overlay",
			postContents: `
{
    "patches": ["mypatchpath.yaml"]
}`,
			expectResult: `{
    "kustomization": "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nbases:\n- ../base\npatchesStrategicMerge:\n- mypatchpath.yaml\n"
}`,
		},
	}

	router := setupRouter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			w := httptest.NewRecorder()

			buf := bytes.NewBufferString(tt.postContents)
			request, err := http.NewRequest("POST", tt.endpoint, buf)
			req.NoError(err)

			router.ServeHTTP(w, request)
			req.Equal(200, w.Code)
			req.Equal(tt.expectResult, w.Body.String())
		})
	}
}

func TestDaemonBytes(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		postContents string
		expectResult []byte
	}{
		{
			name:     "generate tarball",
			endpoint: "/kustomize/bundle",
			postContents: `
{
    "files": [{"filename": "mytestfile.yaml", "contents": "mytestfilecontents"}]
}`,
			expectResult: []byte{0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xca, 0xad, 0x2c, 0x49, 0x2d, 0x2e, 0x49, 0xcb, 0xcc, 0x49, 0xd5, 0xab, 0x4c, 0xcc, 0xcd, 0x61, 0xa0, 0x5, 0x30, 0x30, 0x30, 0x30, 0x30, 0x33, 0x30, 0x0, 0xd3, 0x6, 0x98, 0xb4, 0x81, 0x81, 0x91, 0x11, 0x82, 0xd, 0x12, 0x37, 0x34, 0x32, 0x32, 0x31, 0x60, 0x50, 0x30, 0xa0, 0x89, 0x6b, 0xd0, 0x40, 0x69, 0x71, 0x49, 0x62, 0x11, 0x83, 0x1, 0xc5, 0x76, 0xa1, 0x7b, 0x6e, 0x88, 0x0, 0x44, 0xfc, 0x27, 0xe7, 0xe7, 0x95, 0xa4, 0xe6, 0x95, 0x14, 0xf, 0xb4, 0x8b, 0x46, 0xc1, 0x28, 0x18, 0x5, 0xa3, 0x60, 0x14, 0xd0, 0x3, 0x0, 0x2, 0x0, 0x0, 0xff, 0xff, 0xff, 0x92, 0x4d, 0x7, 0x0, 0x8, 0x0, 0x0},
		},
	}

	router := setupRouter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			w := httptest.NewRecorder()

			buf := bytes.NewBufferString(tt.postContents)
			request, err := http.NewRequest("POST", tt.endpoint, buf)
			req.NoError(err)

			router.ServeHTTP(w, request)
			req.Equal(200, w.Code)
			req.Equal(tt.expectResult, w.Body.Bytes())
		})
	}
}
