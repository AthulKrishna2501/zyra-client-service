package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	pb "github.com/AthulKrishna2501/proto-repo/client"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/repository"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
)

type ClientService struct {
	pb.UnimplementedClientServiceServer
	clientRepo repository.ClientRepository
}

func NewClientService(clientRepo repository.ClientRepository) *ClientService {
	return &ClientService{clientRepo: clientRepo}
}

func (s *ClientService) GetMasterOfCeremony(ctx context.Context, req *pb.MasterOfCeremonyRequest) (*pb.MasterOfCeremonyResponse, error) {
	Amount := 250000

	sessionParams := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("inr"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Master of Ceremony Service"),
					},
					UnitAmount: stripe.Int64(int64(Amount)),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String("http://localhost:3005/success.html"),
		CancelURL:  stripe.String("http://localhost:3005/cancel.html"),
	}

	stripeSession, err := session.New(sessionParams)
	if err != nil {
		return nil, err
	}

	return &pb.MasterOfCeremonyResponse{
		Url: stripeSession.URL,
	}, nil
}

func (s *ClientService) HandleStripeEvent(ctx context.Context, req *pb.StripeWebhookRequest) (*pb.StripeWebhookResponse, error) {
	log.Printf("🔹 Received Stripe Event: %s", req.EventType)
	log.Printf("🔹 Raw Payload: %s", req.Payload)
	var event stripe.Event
	err := json.Unmarshal([]byte(req.Payload), &event)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event: %v", err)
	}

	switch req.EventType {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			return nil, fmt.Errorf("failed to parse session: %v", err)
		}

		clientID, ok := session.Metadata["client_id"]
		if !ok {
			return nil, fmt.Errorf("client ID not found in metadata")
		}

		fmt.Printf("Payment successful! Session ID: %s\n", session.ID)
		err = s.clientRepo.UpdateMasterOfCeremonyStatus(clientID, true)
		if err != nil {
			fmt.Println("Error updating database:", err)
			return nil, fmt.Errorf("failed to update Master of Ceremony status: %v", err)
		}
	default:
		fmt.Printf("ℹReceived unknown event: %s\n", req.EventType)
	}

	return &pb.StripeWebhookResponse{Status: "Success"}, nil
}
