FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build

FROM scratch
COPY --from=builder /app/keycloak-k8s-shim /keycloak-k8s-shim
ENTRYPOINT ["/keycloak-k8s-shim"]
