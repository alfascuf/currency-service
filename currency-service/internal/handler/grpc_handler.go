package handler

import (
	"context"
	"time"

	"github.com/alfascuf/PROD1/currency-service/internal/models"
	"github.com/alfascuf/PROD1/currency-service/internal/service"
	pb "github.com/alfascuf/PROD1/pkg/grpc/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcHandler struct {
	pb.UnimplementedCurrencyServiceServer
	service service.Service // ← Используйте интерфейс Service
}

func NewGrpcHandler(svc service.Service) *GrpcHandler {
	return &GrpcHandler{service: svc}
}

func (h *GrpcHandler) GetRate(ctx context.Context, req *pb.GetRateRequest) (*pb.GetRateResponse, error) {
	// Конвертируем protobuf request в models.GetRateRequest
	serviceReq := &models.GetRateRequest{
		Base:   req.Base,
		Target: req.Target,
		Date:   req.Date,
	}

	// Вызываем метод сервиса
	serviceResp, err := h.service.GetRate(serviceReq)
	if err != nil {
		return &pb.GetRateResponse{
			Error: err.Error(),
		}, nil
	}

	// Если в ответе есть ошибка (бизнес-логика)
	if serviceResp.Error != "" {
		return &pb.GetRateResponse{
			Error: serviceResp.Error,
		}, nil
	}

	// Успешный ответ
	return &pb.GetRateResponse{
		Base:   serviceResp.Base,
		Target: serviceResp.Target,
		Rate:   serviceResp.Rate,
		Date:   serviceResp.Date,
	}, nil
}

func (h *GrpcHandler) GetHistory(ctx context.Context, req *pb.GetHistoryRequest) (*pb.GetHistoryResponse, error) {
	// Конвертируем protobuf request в models.GetHistoryRequest
	serviceReq := &models.GetHistoryRequest{
		Base:      req.Base,
		Target:    req.Target,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	}

	// Вызываем метод сервиса
	serviceResp, err := h.service.GetHistory(serviceReq)
	if err != nil {
		return &pb.GetHistoryResponse{
			Error: err.Error(),
		}, nil
	}

	// Если в ответе есть ошибка (бизнес-логика)
	if serviceResp.Error != "" {
		return &pb.GetHistoryResponse{
			Error: serviceResp.Error,
		}, nil
	}

	// Конвертируем models.ExchangeRate в protobuf ExchangeRate
	data := make([]*pb.ExchangeRate, len(serviceResp.Data))
	for i, rate := range serviceResp.Data {
		data[i] = &pb.ExchangeRate{
			Id:        rate.ID,
			Base:      rate.Base,
			Target:    rate.Target,
			Rate:      rate.Rate,
			Date:      timestamppb.New(rate.Date),
			CreatedAt: timestamppb.New(rate.CreatedAt),
			UpdatedAt: timestamppb.New(rate.UpdatedAt),
		}
	}

	return &pb.GetHistoryResponse{
		Base:   serviceResp.Base,
		Target: serviceResp.Target,
		Data:   data,
	}, nil
}

func (h *GrpcHandler) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status: "OK",
		Time:   time.Now().Format(time.RFC3339),
	}, nil
}
