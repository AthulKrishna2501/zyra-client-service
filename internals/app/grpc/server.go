package grpc

import (
	"net"

	"github.com/AthulKrishna2501/proto-repo/client"

	"github.com/AthulKrishna2501/zyra-client-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/repository"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/services"
	"github.com/AthulKrishna2501/zyra-client-service/internals/logger"
	"google.golang.org/grpc"
)

func StartgRPCServer(ClientRepo repository.ClientRepository, log logger.Logger, cfg config.Config) error {
	go func() {
		lis, err := net.Listen("tcp", ":5002")
		if err != nil {
			log.Error("Failed to listen on port 5002: %v", err)
			return
		}

		grpcServer := grpc.NewServer(
			grpc.MaxRecvMsgSize(1024*1024*100),
			grpc.MaxSendMsgSize(1024*1024*100),
		)
		ClientService := services.NewClientService(ClientRepo, cfg, log)
		client.RegisterClientServiceServer(grpcServer, ClientService)

		log.Info("gRPC Server started on port 5002")
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("Failed to serve gRPC: %v", err)
		}
	}()

	return nil

}
