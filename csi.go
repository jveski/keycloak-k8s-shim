package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type ClientSecretGetter interface {
	Fetch(ctx context.Context, clientID string) ([]byte, error)
}

type Server struct {
	csi.UnimplementedIdentityServer
	csi.UnimplementedNodeServer

	Getter ClientSecretGetter
}

func (s *Server) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if req.VolumeContext == nil || req.VolumeContext["clientID"] == "" {
		return nil, status.Error(codes.FailedPrecondition, "must specify clientID in the volume context")
	}
	clientID := req.VolumeContext["clientID"]

	secret, err := s.Getter.Fetch(ctx, clientID)
	if err != nil {
		log.Printf("error while getting secret for volume %q: %s", req.VolumeId, err)
		return nil, err
	}

	err = os.MkdirAll(req.TargetPath, 0777)
	if err != nil {
		log.Printf("error while creating mount dir for volume %q: %s", req.VolumeId, err)
		return nil, err
	}

	err = os.WriteFile(path.Join(req.TargetPath, "client-id"), []byte(clientID), 0444)
	if err != nil {
		log.Printf("error while writing client ID for volume %q: %s", req.VolumeId, err)
		return nil, fmt.Errorf("writing file: %w", err)
	}

	err = os.WriteFile(path.Join(req.TargetPath, "client-secret"), secret, 0444)
	if err != nil {
		log.Printf("error while writing secret for volume %q: %s", req.VolumeId, err)
		return nil, fmt.Errorf("writing file: %w", err)
	}

	log.Printf("mounted %s", req.VolumeId)
	return &csi.NodePublishVolumeResponse{}, nil
}

func (s *Server) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	for _, file := range []string{
		path.Join(req.TargetPath, "client-id"),
		path.Join(req.TargetPath, "client-secret"),
	} {
		if err := os.RemoveAll(file); err != nil {
			log.Printf("error while removing file %q: %s", req.VolumeId, err)
			return nil, err
		}
	}

	log.Printf("unmounted %s", req.VolumeId)
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (s *Server) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	resp := &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{{
			Type: &csi.PluginCapability_Service_{
				Service: &csi.PluginCapability_Service{
					Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
				},
			},
		}},
	}
	return resp, nil
}

func (s *Server) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	resp := &csi.GetPluginInfoResponse{
		Name:          "identity.keycloak.org",
		VendorVersion: "1.0.0",
	}
	return resp, nil
}

func (s *Server) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{Ready: &wrapperspb.BoolValue{Value: true}}, nil
}

func (s *Server) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	resp := &csi.NodeGetInfoResponse{
		NodeId: os.Getenv("NODE_ID"),
	}
	return resp, nil
}

func (s *Server) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	resp := &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{},
	}
	return resp, nil
}
