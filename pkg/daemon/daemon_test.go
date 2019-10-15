package daemon

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExistingPatch(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			router := setupRouter()
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