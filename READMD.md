# Keycloak Client Secret CSI Driver

A CSI driver to expose Keycloak client secrets, intended for providing Keycloak identities to Kubernetes pods.


## Usage

- `--uds-path`: Path to the CSI driver's gRPC UDS
- `--keycloak-url`: Base URL of Keycloak
- `--keycloak-username`: Username to present when getting access tokens
- `--keycloak-password-file`: Path to a file with the corresponding password
- `NODE_ID`: Set to Kubernetes node name
