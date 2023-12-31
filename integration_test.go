package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMountIntegration(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.String(), "/realms/test-realm/protocol/openid-connect/token"):
			io.WriteString(w, `{ "access_token": "test-token", "expires_in": 1 }`)

		case strings.HasSuffix(r.URL.String(), "/admin/realms/test-realm/clients?clientId=test-client-id"):
			io.WriteString(w, `[{ "id": "test-client-uuid" }]`)

		case strings.HasSuffix(r.URL.String(), "/admin/realms/test-realm/clients/test-client-uuid/client-secret"):
			io.WriteString(w, `{ "value": "test-client-secret" }`)
		}
	}))
	defer svr.Close()

	pwordFile := path.Join(t.TempDir(), "password")
	err := os.WriteFile(pwordFile, []byte("test-password"), 0644)
	require.NoError(t, err)

	kc, err := NewKeycloak(svr.URL+"/base", "test-realm", "test-username", pwordFile, time.Second*5)
	require.NoError(t, err)

	s := &Server{Getter: kc}
	ctx := context.Background()

	// Mount a secret
	targetPath := path.Join(t.TempDir(), "test", "target-path")
	_, err = s.NodePublishVolume(ctx, &csi.NodePublishVolumeRequest{
		VolumeId:      "test-volume-id",
		TargetPath:    targetPath,
		VolumeContext: map[string]string{"clientID": "test-client-id"},
	})
	require.NoError(t, err)

	// Confirm files were created
	actualClientID, err := os.ReadFile(path.Join(targetPath, "client-id"))
	require.NoError(t, err)
	assert.Equal(t, "test-client-id", string(actualClientID))

	actualClientSecret, err := os.ReadFile(path.Join(targetPath, "client-secret"))
	require.NoError(t, err)
	assert.Equal(t, "test-client-secret", string(actualClientSecret))

	// Unmount it
	_, err = s.NodeUnpublishVolume(ctx, &csi.NodeUnpublishVolumeRequest{
		VolumeId:   "test-volume-id",
		TargetPath: targetPath,
	})
	require.NoError(t, err)
	assert.NoFileExists(t, path.Join(targetPath, "client-id"))
	assert.NoFileExists(t, path.Join(targetPath, "client-secret"))
}
