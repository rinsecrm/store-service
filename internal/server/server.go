package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rinsecrm/store-service/core/logging"
	"github.com/rinsecrm/store-service/internal/data"
	"github.com/rinsecrm/store-service/internal/metrics"
	"github.com/rinsecrm/store-service/internal/tracing"
	pb "github.com/rinsecrm/store-service/proto/go"
)

// StoreServiceServer implements the StoreService gRPC interface
type StoreServiceServer struct {
	pb.UnimplementedStoreServiceServer
	store data.StoreInterface
}

// NewStoreServiceServer creates a new server instance
func NewStoreServiceServer(store data.StoreInterface) *StoreServiceServer {
	return &StoreServiceServer{
		store: store,
	}
}

// CreateItem creates a new store item
func (s *StoreServiceServer) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.CreateItemResponse, error) {
	// Start custom span for business logic
	ctx, span := tracing.StartSpan(ctx, "store.create_item")
	defer span.End()

	start := time.Now()

	// Validate request
	if req.TenantId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "tenant_id must be positive")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Price < 0 {
		return nil, status.Error(codes.InvalidArgument, "price cannot be negative")
	}

	// Convert proto enums to data types
	category := protoToDataCategory(req.Category)

	item, err := s.store.CreateItem(
		ctx,
		req.TenantId,
		req.Name,
		req.Description,
		req.Price,
		category,
		req.Sku,
		req.InventoryCount,
		req.Tags,
		req.CreatedBy,
	)
	if err != nil {
		// Record error metrics
		metrics.RecordStoreOperationError("create")

		logging.WithError(err).WithFields(logging.Fields{
			"tenant_id": req.TenantId,
			"name":      req.Name,
		}).Error("Failed to create item")
		return nil, status.Error(codes.Internal, "failed to create item")
	}

	duration := time.Since(start)

	// Record business metrics
	metrics.RecordStoreOperation("create")
	metrics.RecordStoreOperationDuration("create", float64(duration)/float64(time.Second))

	logging.WithFields(logging.Fields{
		"tenant_id": req.TenantId,
		"item_id":   item.ItemID,
		"duration":  duration,
	}).Info("Item created via gRPC")

	return &pb.CreateItemResponse{
		Item: dataToProtoItem(item),
	}, nil
}

// GetItem retrieves an item by ID
func (s *StoreServiceServer) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.GetItemResponse, error) {
	// Start custom span for business logic
	ctx, span := tracing.StartSpan(ctx, "store.get_item")
	defer span.End()

	start := time.Now()

	if req.TenantId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "tenant_id must be positive")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	item, err := s.store.GetItem(ctx, req.TenantId, req.Id)
	if err != nil {
		if err == data.ErrItemNotFound {
			return nil, status.Error(codes.NotFound, "item not found")
		}
		logging.WithError(err).WithFields(logging.Fields{
			"tenant_id": req.TenantId,
			"item_id":   req.Id,
		}).Error("Failed to get item")
		return nil, status.Error(codes.Internal, "failed to get item")
	}

	logging.WithFields(logging.Fields{
		"tenant_id": req.TenantId,
		"item_id":   req.Id,
		"duration":  time.Since(start),
	}).Debug("Item retrieved via gRPC")

	return &pb.GetItemResponse{
		Item: dataToProtoItem(item),
	}, nil
}

// UpdateItem updates an existing item
func (s *StoreServiceServer) UpdateItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.UpdateItemResponse, error) {
	// Start custom span for business logic
	ctx, span := tracing.StartSpan(ctx, "store.update_item")
	defer span.End()

	start := time.Now()

	if req.TenantId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "tenant_id must be positive")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Price < 0 {
		return nil, status.Error(codes.InvalidArgument, "price cannot be negative")
	}

	category := protoToDataCategory(req.Category)
	status := protoToDataStatus(req.Status)

	item, err := s.store.UpdateItem(
		ctx,
		req.TenantId,
		req.Id,
		req.Name,
		req.Description,
		req.Price,
		category,
		status,
		req.Sku,
		req.InventoryCount,
		req.Tags,
		req.UpdatedBy,
	)
	if err != nil {
		if err == data.ErrItemNotFound {
			return nil, status.Error(codes.NotFound, "item not found")
		}
		logging.WithError(err).WithFields(logging.Fields{
			"tenant_id": req.TenantId,
			"item_id":   req.Id,
		}).Error("Failed to update item")
		return nil, status.Error(codes.Internal, "failed to update item")
	}

	logging.WithFields(logging.Fields{
		"tenant_id": req.TenantId,
		"item_id":   req.Id,
		"duration":  time.Since(start),
	}).Info("Item updated via gRPC")

	return &pb.UpdateItemResponse{
		Item: dataToProtoItem(item),
	}, nil
}

// DeleteItem removes an item (soft delete)
func (s *StoreServiceServer) DeleteItem(ctx context.Context, req *pb.DeleteItemRequest) (*pb.DeleteItemResponse, error) {
	// Start custom span for business logic
	ctx, span := tracing.StartSpan(ctx, "store.delete_item")
	defer span.End()

	start := time.Now()

	if req.TenantId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "tenant_id must be positive")
	}
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	err := s.store.DeleteItem(ctx, req.TenantId, req.Id)
	if err != nil {
		if err == data.ErrItemNotFound {
			return nil, status.Error(codes.NotFound, "item not found")
		}
		logging.WithError(err).WithFields(logging.Fields{
			"tenant_id": req.TenantId,
			"item_id":   req.Id,
		}).Error("Failed to delete item")
		return nil, status.Error(codes.Internal, "failed to delete item")
	}

	logging.WithFields(logging.Fields{
		"tenant_id": req.TenantId,
		"item_id":   req.Id,
		"duration":  time.Since(start),
	}).Info("Item deleted via gRPC")

	return &pb.DeleteItemResponse{
		Success: true,
	}, nil
}

// ListItems lists items with filtering and pagination
func (s *StoreServiceServer) ListItems(ctx context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	// Start custom span for business logic
	ctx, span := tracing.StartSpan(ctx, "store.list_items")
	defer span.End()

	start := time.Now()

	if req.TenantId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "tenant_id must be positive")
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 100 // Default page size
	}
	if pageSize > 1000 {
		pageSize = 1000 // Max page size
	}

	category := protoToDataCategory(req.Category)
	status := protoToDataStatus(req.Status)

	items, nextPageToken, totalCount, err := s.store.ListItems(
		ctx,
		req.TenantId,
		category,
		status,
		req.SearchQuery,
		pageSize,
		req.PageToken,
	)
	if err != nil {
		logging.WithError(err).WithFields(logging.Fields{
			"tenant_id": req.TenantId,
		}).Error("Failed to list items")
		return nil, status.Error(codes.Internal, "failed to list items")
	}

	var protoItems []*pb.Item
	for _, item := range items {
		protoItems = append(protoItems, dataToProtoItem(item))
	}

	logging.WithFields(logging.Fields{
		"tenant_id":   req.TenantId,
		"items_count": len(items),
		"duration":    time.Since(start),
	}).Debug("Items listed via gRPC")

	return &pb.ListItemsResponse{
		Items:         protoItems,
		NextPageToken: nextPageToken,
		TotalCount:    totalCount,
	}, nil
}

// UpdateInventory updates the inventory count for an item
func (s *StoreServiceServer) UpdateInventory(ctx context.Context, req *pb.UpdateInventoryRequest) (*pb.UpdateInventoryResponse, error) {
	// Start custom span for business logic
	ctx, span := tracing.StartSpan(ctx, "store.update_inventory")
	defer span.End()

	start := time.Now()

	if req.TenantId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "tenant_id must be positive")
	}
	if req.ItemId == "" {
		return nil, status.Error(codes.InvalidArgument, "item_id is required")
	}

	item, previousCount, err := s.store.UpdateInventory(
		ctx,
		req.TenantId,
		req.ItemId,
		req.QuantityChange,
		req.Reason,
		req.UpdatedBy,
	)
	if err != nil {
		if err == data.ErrItemNotFound {
			return nil, status.Error(codes.NotFound, "item not found")
		}
		logging.WithError(err).WithFields(logging.Fields{
			"tenant_id": req.TenantId,
			"item_id":   req.ItemId,
		}).Error("Failed to update inventory")
		return nil, status.Error(codes.Internal, "failed to update inventory")
	}

	logging.WithFields(logging.Fields{
		"tenant_id":       req.TenantId,
		"item_id":         req.ItemId,
		"quantity_change": req.QuantityChange,
		"previous_count":  previousCount,
		"duration":        time.Since(start),
	}).Info("Inventory updated via gRPC")

	return &pb.UpdateInventoryResponse{
		Item:          dataToProtoItem(item),
		PreviousCount: previousCount,
	}, nil
}

// Helper functions for converting between proto and data types

func protoToDataCategory(category pb.ItemCategory) data.ItemCategory {
	switch category {
	case pb.ItemCategory_ITEM_CATEGORY_ELECTRONICS:
		return data.ItemCategoryElectronics
	case pb.ItemCategory_ITEM_CATEGORY_CLOTHING:
		return data.ItemCategoryClothing
	case pb.ItemCategory_ITEM_CATEGORY_BOOKS:
		return data.ItemCategoryBooks
	case pb.ItemCategory_ITEM_CATEGORY_HOME:
		return data.ItemCategoryHome
	case pb.ItemCategory_ITEM_CATEGORY_SPORTS:
		return data.ItemCategorySports
	default:
		return data.ItemCategoryUnspecified
	}
}

func dataToProtoCategory(category data.ItemCategory) pb.ItemCategory {
	switch category {
	case data.ItemCategoryElectronics:
		return pb.ItemCategory_ITEM_CATEGORY_ELECTRONICS
	case data.ItemCategoryClothing:
		return pb.ItemCategory_ITEM_CATEGORY_CLOTHING
	case data.ItemCategoryBooks:
		return pb.ItemCategory_ITEM_CATEGORY_BOOKS
	case data.ItemCategoryHome:
		return pb.ItemCategory_ITEM_CATEGORY_HOME
	case data.ItemCategorySports:
		return pb.ItemCategory_ITEM_CATEGORY_SPORTS
	default:
		return pb.ItemCategory_ITEM_CATEGORY_UNSPECIFIED
	}
}

func protoToDataStatus(status pb.ItemStatus) data.ItemStatus {
	switch status {
	case pb.ItemStatus_ITEM_STATUS_ACTIVE:
		return data.ItemStatusActive
	case pb.ItemStatus_ITEM_STATUS_INACTIVE:
		return data.ItemStatusInactive
	case pb.ItemStatus_ITEM_STATUS_OUT_OF_STOCK:
		return data.ItemStatusOutOfStock
	case pb.ItemStatus_ITEM_STATUS_DISCONTINUED:
		return data.ItemStatusDiscontinued
	default:
		return data.ItemStatusUnspecified
	}
}

func dataToProtoStatus(status data.ItemStatus) pb.ItemStatus {
	switch status {
	case data.ItemStatusActive:
		return pb.ItemStatus_ITEM_STATUS_ACTIVE
	case data.ItemStatusInactive:
		return pb.ItemStatus_ITEM_STATUS_INACTIVE
	case data.ItemStatusOutOfStock:
		return pb.ItemStatus_ITEM_STATUS_OUT_OF_STOCK
	case data.ItemStatusDiscontinued:
		return pb.ItemStatus_ITEM_STATUS_DISCONTINUED
	default:
		return pb.ItemStatus_ITEM_STATUS_UNSPECIFIED
	}
}

func dataToProtoItem(item data.Item) *pb.Item {
	return &pb.Item{
		Id:             item.ItemID,
		TenantId:       item.TenantID,
		Name:           item.Name,
		Description:    item.Description,
		Price:          item.Price,
		Category:       dataToProtoCategory(item.Category),
		Status:         dataToProtoStatus(item.Status),
		Sku:            item.SKU,
		InventoryCount: item.InventoryCount,
		Tags:           item.Tags,
		CreatedAt:      timestamppb.New(item.CreatedAt),
		UpdatedAt:      timestamppb.New(item.UpdatedAt),
		CreatedBy:      item.CreatedBy,
		UpdatedBy:      item.UpdatedBy,
	}
}
