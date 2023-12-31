# Keycloak Client Secret CSI Driver

A CSI driver to expose Keycloak client secrets, intended for providing Keycloak identities to Kubernetes pods.


## Usage

- `--uds-path`: Path to the CSI driver's gRPC UDS
- `--keycloak-url`: Base URL of Keycloak
- `--keycloak-username`: Username to present when getting access tokens
- `--keycloak-password-file`: Path to a file with the corresponding password
- `NODE_ID`: Set to Kubernetes node name

Example daemonset:

```yaml
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
        livenessProbe:
          httpGet:
            path: /healthz
            port: 9090
          initialDelaySeconds: 5
          timeoutSeconds: 5
        volumeMounts:
          - name: plugin-dir
            mountPath: /csi
          - name: registration-dir
            mountPath: /registration

      - name: csi-driver
        image: "ghcr.io/jveski/keycloak-k8s-shim:main-41b56d8" # or latest release tag
        args:
          - --keycloak-url=https://your-keycloak-instance
          - --keycloak-username=admin # or another username
          - --keycloak-password-path=/etc/keycloak/password
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
            mountPath: "/etc/keycloak"
```
