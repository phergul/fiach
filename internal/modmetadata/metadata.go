package modmetadata

import (
	"context"
	"errors"
	"fmt"
)

type Metadata struct {
	FileCount      *int64
	DirectoryCount *int64
	TotalSizeBytes *int64
	JSON           *string
	Version        *string
	Author         *string
	Description    *string
	SourceURL      *string
}

type ParseInput struct {
	ManagedPath string
	GameID      int64
	SourceType  string
}

type Parser interface {
	Parse(context.Context, ParseInput) (Metadata, error)
}

type Registry struct {
	parsers []Parser
}

func NewRegistry(parsers ...Parser) *Registry {
	return &Registry{
		parsers: append([]Parser{}, parsers...),
	}
}

func DefaultRegistry() *Registry {
	return NewRegistry(InventoryParser{})
}

func (r *Registry) Parse(ctx context.Context, input ParseInput) (metadata Metadata, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("parse mod metadata: %w", err)
		}
	}()

	if r == nil || len(r.parsers) == 0 {
		return Metadata{}, errors.New("metadata parser registry is not configured")
	}

	for _, parser := range r.parsers {
		if parser == nil {
			continue
		}

		parsedMetadata, err := parser.Parse(ctx, input)
		if err != nil {
			return Metadata{}, err
		}
		metadata = mergeMetadata(metadata, parsedMetadata)
	}

	return metadata, nil
}

func mergeMetadata(base Metadata, next Metadata) Metadata {
	if next.FileCount != nil {
		base.FileCount = next.FileCount
	}
	if next.DirectoryCount != nil {
		base.DirectoryCount = next.DirectoryCount
	}
	if next.TotalSizeBytes != nil {
		base.TotalSizeBytes = next.TotalSizeBytes
	}
	if next.JSON != nil {
		base.JSON = next.JSON
	}
	if next.Version != nil {
		base.Version = next.Version
	}
	if next.Author != nil {
		base.Author = next.Author
	}
	if next.Description != nil {
		base.Description = next.Description
	}
	if next.SourceURL != nil {
		base.SourceURL = next.SourceURL
	}

	return base
}

func int64Ptr(value int64) *int64 {
	return &value
}
