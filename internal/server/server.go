package server

import (
	"context"
	"fmt"
	"net"

	"github.com/rrb115/vdcs/internal/node"
	vdcspb "github.com/rrb115/vdcs/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the VDCS gRPC service.
type Server struct {
	vdcspb.UnimplementedVDCSServer
	node *node.Node
}

// NewServer creates a new VDCS gRPC server.
func NewServer(n *node.Node) *Server {
	return &Server{node: n}
}

func (s *Server) ProposeEntry(ctx context.Context, req *vdcspb.ConfigEntry) (*vdcspb.ProposeResponse, error) {
	if err := s.node.ProposeEntry(req); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to propose entry: %v", err)
	}
	return &vdcspb.ProposeResponse{}, nil
}

func (s *Server) GetLatestRoot(ctx context.Context, req *vdcspb.Empty) (*vdcspb.ConfigState, error) {
	ver, root, headHash := s.node.GetLatestRoot()
	return &vdcspb.ConfigState{
		Version:       ver,
		StateRoot:     root,
		LastEntryHash: headHash,
	}, nil
}

func (s *Server) GetProof(ctx context.Context, req *vdcspb.GetProofRequest) (*vdcspb.GetProofResponse, error) {
	proof, err := s.node.GetProof(req.Key)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "proof not found: %v", err)
	}
	return &vdcspb.GetProofResponse{
		Key:       proof.Key,
		ValueHash: proof.ValueHash,
		Siblings:  proof.Siblings,
		IsLeft:    proof.IsLeft,
	}, nil
}

// Start starts the gRPC server on the given port.
func (s *Server) Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	vdcspb.RegisterVDCSServer(grpcServer, s)

	// Serving...
	return grpcServer.Serve(lis)
}
