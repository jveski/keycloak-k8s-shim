# Keycloak Client Secret CSI Driver

A CSI driver to expose Keycloak client secrets, intended for providing Keycloak identities to Kubernetes pods.


## Usage

- `--uds-path`: Path to the CSI driver's gRPC UDS
- `--keycloak-url`: Base URL of Keycloak
- `--keycloak-username`: Username to present when getting access tokens
- `--keycloak-password-file`: Path to a file with the corresponding password
- `NODE_ID`: Set to Kubernetes node name

Example manifest:

```yaml
apiVersion: storage.k8s.io/v1
kind: CSIDriver
metadata:
  name: identity.keycloak.org
spec:
  volumeLifecycleModes:
    - Ephemeral

---

apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: keycloak-csi-driver
spec:
  selector:
    matchLabels:
      name: keycloak-csi-driver
  template:
    metadata:
      labels:
        name: keycloak-csi-driver
    spec:
      terminationGracePeriodSeconds: 30

      volumes:
        - name: registration-dir
          hostPath:
            path: /var/lib/kubelet/plugins_registry/
            type: Directory
        - name: plugin-dir
          hostPath:
            path: /var/lib/kubelet/plugins/identity.keycloak.org/
            type: DirectoryOrCreate
        - name: keycloak-password
          secret:
            secretName: keycloak-admin

      containers:
      - name: csi-driver-registrar
        image: k8s.gcr.io/sig-storage/csi-node-driver-registrar:v2.9.3
        args:
          - "--csi-address=/csi/csi.sock"
          - "--kubelet-registration-path=/var/lib/kubelet/plugins/identity.keycloak.org/csi.sock"
          - "--health-port=9809"
        volumeMounts:
          - name: plugin-dir
            mountPath: /csi
          - name: registration-dir
            mountPath: /registration

      - name: csi-driver
        image: "ghcr.io/jveski/keycloak-k8s-shim:USE_LATEST_RELEASE_TAG"
        args:
          - --keycloak-url=https://your-keycloak-instance
        env:
          - name: NODE_ID
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
        volumeMounts:
          - name: plugin-dir
            mountPath: /csi
          - name: keycloak-password
            readOnly: true
            mountPath: "/etc/keycloak" # assumes the secret contains the key "password"
```
