package builtin

import (
	"context"

	api "github.com/akuity/kargo/api/v1alpha1"
	conversion "github.com/akuity/kargo/internal/proto/grpc"
	freightv1 "github.com/akuity/kargo/pkg/generated/freight/v1"
	"github.com/akuity/kargo/pkg/promotion"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcRunner struct {
	server string
}

// grpcRunner explicitly implements promotion.StepRunner
var _ promotion.StepRunner = &grpcRunner{}

func (gr *grpcRunner) Name() string {
	return "grpc-runner"
}

func (gr *grpcRunner) Run(ctx context.Context, stepCtx *promotion.StepContext) (promotion.StepResult, error) {
	// TODO: wire server endpoint to configs
	conn, err := grpc.NewClient("localhost:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return promotion.StepResult{Status: api.PromotionStepStatusErrored}, err
	}
	req := &freightv1.ProcessFreightRequest{
		Context: &freightv1.PromotionContext{
			Project:   stepCtx.Project,
			Stage:     stepCtx.Stage,
			Promotion: stepCtx.Promotion,
			StepAlias: stepCtx.Alias,
		},
		Freight:       conversion.KargoFreightCollectionToProto(stepCtx.Freight),
		TargetFreight: conversion.KargoFreightReferenceToProto(stepCtx.TargetFreightRef),
	}

	client := freightv1.NewFreightProcessorServiceClient(conn)
	resp, err := client.ProcessFreight(ctx, req)
	if err != nil {
		return promotion.StepResult{Status: api.PromotionStepStatusErrored}, err
	}

	if resp.ModifiedFreight != nil {
		modifiedKargoFreight := conversion.ProtoFreightCollectionToKargo(resp.ModifiedFreight)
		stepCtx.Freight = modifiedKargoFreight
	}

	return promotion.StepResult{Status: api.PromotionStepStatusSucceeded}, nil
}
