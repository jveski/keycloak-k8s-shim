# Keycloak Client Secret CSI Driver

A CSI driver to expose Keycloak client secrets, intended for providing Keycloak identities to Kubernetes pods.


## Usage

See example.yaml.

- `--uds-path`: Path to the CSI driver's gRPC UDS
- `--keycloak-url`: Base URL of Keycloak
- `--keycloak-username`: Username to present when getting access tokens
- `--keycloak-password-file`: Path to a file with the corresponding password
- `NODE_ID`: Set to Kubernetes node name


## Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keycloak-test
  labels:
    app: keycloak-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: keycloak-test
  template:
    metadata:
      labels:
        app: keycloak-test
    spec:
      containers:
        - name: keycloak-test
          image: nginx
          volumeMounts:
            - name: keycloak-creds
              mountPath: /var/lib/keycloak
      volumes:
        - name: keycloak-creds
          csi:
            driver: identity.keycloak.org
            volumeAttributes:
              clientID: name-of-your-client
```
