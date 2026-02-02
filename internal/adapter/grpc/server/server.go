package server

import (
	"context"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/seu-repo/sigec-ve/api/proto/device/v1"
	"github.com/seu-repo/sigec-ve/internal/ports"
)

type GRPCServer struct {
	server *grpc.Server
	log    *zap.Logger
}

type DeviceGrpcService struct {
	pb.UnimplementedDeviceServiceServer
	deviceService ports.DeviceService
	txService     ports.TransactionService // Assuming it needs this or separate server
	log           *zap.Logger
}

func NewGRPCServer(deviceService ports.DeviceService, txService ports.TransactionService, log *zap.Logger) *GRPCServer {
	s := grpc.NewServer()

	// Register services
	pb.RegisterDeviceServiceServer(s, &DeviceGrpcService{
		deviceService: deviceService,
		txService:     txService,
		log:           log,
	})

	// Enable reflection for debugging (e.g. grpcurl)
	reflection.Register(s)

	return &GRPCServer{
		server: s,
		log:    log,
	}
}

func (s *GRPCServer) Serve(lis net.Listener) error {
	return s.server.Serve(lis)
}

func (s *GRPCServer) Stop() {
	s.server.GracefulStop()
}

// Implement handlers
func (s *DeviceGrpcService) GetDevice(ctx context.Context, req *pb.GetDeviceRequest) (*pb.GetDeviceResponse, error) {
	device, err := s.deviceService.GetDevice(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, nil // Or Status.NotFound
	}

	// Map domain to proto
	return &pb.GetDeviceResponse{
		Device: &pb.Device{
			Id:     device.ID,
			Vendor: device.Vendor,
			Model:  device.Model,
			// ... map other fields
		},
	}, nil
}

// ... other methods
func (s *DeviceGrpcService) ListDevices(ctx context.Context, req *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {
	return &pb.ListDevicesResponse{}, nil
}

func (s *DeviceGrpcService) UpdateDeviceStatus(ctx context.Context, req *pb.UpdateDeviceStatusRequest) (*pb.UpdateDeviceStatusResponse, error) {
	return &pb.UpdateDeviceStatusResponse{}, nil
}

func (s *DeviceGrpcService) StreamDeviceEvents(req *pb.StreamDeviceEventsRequest, stream pb.DeviceService_StreamDeviceEventsServer) error {
	return nil
}
