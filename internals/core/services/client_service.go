package services

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/AthulKrishna2501/proto-repo/client"
	"github.com/AthulKrishna2501/zyra-client-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/repository"
	"github.com/AthulKrishna2501/zyra-client-service/internals/logger"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
)

type ClientService struct {
	pb.UnimplementedClientServiceServer
	clientRepo repository.ClientRepository
	config     config.Config
	log        logger.Logger
}

func NewClientService(clientRepo repository.ClientRepository, cfg config.Config, logger logger.Logger) *ClientService {
	return &ClientService{clientRepo: clientRepo, config: cfg, log: logger}
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
		SuccessURL: stripe.String(s.config.STRIPE_SUCCESS_URL),
		CancelURL:  stripe.String(s.config.STRIPE_CANCEL_URL),
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
	s.log.Info("Received Stripe Event: %s", req.EventType)
	s.log.Info("Raw Payload: %s", req.Payload)
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

		s.log.Info("Payment successful! Session ID: %s\n", session.ID)
		err = s.clientRepo.UpdateMasterOfCeremonyStatus(clientID, true)
		if err != nil {
			return nil, fmt.Errorf("failed to update Master of Ceremony status: %v", err)
		}
	default:
		s.log.Info("Received unknown event: %s\n", req.EventType)
	}

	return &pb.StripeWebhookResponse{Status: "Success"}, nil
}

func (s *ClientService) VerifyPayment(ctx context.Context, req *pb.VerifyPaymentRequest) (*pb.VerifyPaymentResponse, error) {
	const amount = 2500
	stripeSession, err := session.Get(req.SessionId, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %v", err)
	}

	if stripeSession.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
		err := s.clientRepo.CreditAdminWallet(amount, s.config.ADMIN_EMAIL)
		if err != nil {
			return nil, fmt.Errorf("failed to update createAdminWallet %v", err.Error())
		}
		return &pb.VerifyPaymentResponse{
			Success: true,
			Message: "Payment successful",
		}, nil
	} else {
		return &pb.VerifyPaymentResponse{
			Success: true,
			Message: "The payment was not completed or was canceled.",
		}, nil
	}
}
