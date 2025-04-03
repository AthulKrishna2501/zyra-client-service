package main

import (
	"github.com/AthulKrishna2501/zyra-client-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-client-service/internals/app/grpc"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/database"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/repository"
	"github.com/AthulKrishna2501/zyra-client-service/internals/logger"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v76"
)

func main() {
	log := logger.NewLogrusLogger()

	configEnv, err := config.LoadConfig()
	if err != nil {
		log.Error("Error in config .env: %v", err)
		return
	}

	stripe.Key = configEnv.STRIPE_SECRET_KEY
	db := database.ConnectDatabase(configEnv)
	if db == nil {
		log.Error("Failed to connect to database")
		return
	}

	ClientRepo := repository.NewClientRepository(db)

	err = grpc.StartgRPCServer(ClientRepo, log, configEnv)

	if err != nil {
		log.Error("Faile to start gRPC server", err)
		return
	}

	router := gin.Default()
	log.Info("HTTP Server started on port 3005")
	router.Static("/", "../internals/static")
	router.Run(":3005")

}
