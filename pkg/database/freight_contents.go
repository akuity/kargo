package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// FreightContents holds the separate artifact rows belonging to one Freight.
type FreightContents struct {
	Commits   []UpsertFreightCommitParams
	Images    []UpsertFreightImageParams
	Charts    []UpsertFreightChartParams
	Artifacts []UpsertFreightArtifactParams
}

// FreightUpsert writes the identity and all artifact rows in one transaction.
type FreightUpsert struct {
	UpsertFreightParams
	FreightContents
}

// FreightSnapshot contains a Freight row and its artifacts from the same snapshot.
type FreightSnapshot struct {
	Freight
	FreightContents
}

func replaceFreightContents(ctx context.Context, q *Queries, id string, contents FreightContents) error {
	for _, remove := range []func(context.Context, string) error{
		q.DeleteFreightCommits, q.DeleteFreightImages, q.DeleteFreightCharts, q.DeleteFreightArtifacts,
	} {
		if err := remove(ctx, id); err != nil {
			return fmt.Errorf("error replacing freight artifacts: %w", err)
		}
	}
	for _, row := range contents.Commits {
		row.FreightID = id
		if err := q.UpsertFreightCommit(ctx, row); err != nil {
			return fmt.Errorf("error upserting freight commit: %w", err)
		}
	}
	for _, row := range contents.Images {
		row.FreightID = id
		if err := q.UpsertFreightImage(ctx, row); err != nil {
			return fmt.Errorf("error upserting freight image: %w", err)
		}
	}
	for _, row := range contents.Charts {
		row.FreightID = id
		if err := q.UpsertFreightChart(ctx, row); err != nil {
			return fmt.Errorf("error upserting freight chart: %w", err)
		}
	}
	for _, row := range contents.Artifacts {
		row.FreightID = id
		if err := q.UpsertFreightArtifact(ctx, row); err != nil {
			return fmt.Errorf("error upserting freight artifact: %w", err)
		}
	}
	return nil
}

func (s *store) ListFreight(ctx context.Context) ([]FreightSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	var result []FreightSnapshot
	// A single database snapshot prevents concurrent writes from mixing a Freight
	// identity with an older or newer collection of artifact rows during Diff.
	err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly,
	}, func(tx pgx.Tx) error {
		var readErr error
		result, readErr = listFreightContents(ctx, s.queries.WithTx(tx))
		return readErr
	})
	return result, err
}

func listFreightContents(ctx context.Context, q *Queries) ([]FreightSnapshot, error) {
	rows, err := q.ListFreight(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]FreightSnapshot, len(rows))
	byID := make(map[string]*FreightSnapshot, len(rows))
	for i, row := range rows {
		result[i].Freight = row
		byID[row.ID] = &result[i]
	}
	commits, err := q.ListFreightCommits(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range commits {
		freight := byID[row.FreightID]
		freight.Commits = append(freight.Commits, UpsertFreightCommitParams(row))
	}
	images, err := q.ListFreightImages(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range images {
		freight := byID[row.FreightID]
		freight.Images = append(freight.Images, UpsertFreightImageParams(row))
	}
	charts, err := q.ListFreightCharts(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range charts {
		freight := byID[row.FreightID]
		freight.Charts = append(freight.Charts, UpsertFreightChartParams(row))
	}
	artifacts, err := q.ListFreightArtifacts(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range artifacts {
		freight := byID[row.FreightID]
		freight.Artifacts = append(freight.Artifacts, UpsertFreightArtifactParams(row))
	}
	return result, nil
}
