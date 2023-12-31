package main

import (
	"flag"
	"net"
	"os"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	var (
		sockAddr                 = flag.String("uds-path", "/csi/csi.sock", "path to the uds")
		keycloakTimeout          = flag.Duration("keycloak-timeout", time.Second*10, "timeout for requests to Keycloak")
		keycloakURL              = flag.String("keycloak-url", "", "URL of the Keycloak instance")
		keycloakRealm            = flag.String("keycloak-realm", "master", "Keycloak realm")
		keycloakClientID         = flag.String("keycloak-client-id", "k8s-csi-driver", "The controller's identity")
		keycloakClientSecretFile = flag.String("keycloak-client-secret-file", "/etc/keycloak/password", "Path to a file with --keycloak-client-id's secret")
	)
	flag.Parse()

	if _, err := os.Stat(*sockAddr); !os.IsNotExist(err) {
		if err := os.RemoveAll(*sockAddr); err != nil {
			return err
		}
	}

	listener, err := net.Listen("unix", *sockAddr)
	if err != nil {
		return err
	}

	kc, err := NewKeycloak(*keycloakURL, *keycloakRealm, *keycloakClientID, *keycloakClientSecretFile, *keycloakTimeout)
	if err != nil {
		return err
	}

	impl := &Server{Getter: kc}
	server := grpc.NewServer()
	csi.RegisterIdentityServer(server, impl)
	csi.RegisterNodeServer(server, impl)

	return server.Serve(listener)
}
