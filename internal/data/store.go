package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/rinsecrm/store-service/core/logging"
)

// ErrItemNotFound is returned when an item is not found
var ErrItemNotFound = errors.New("item not found")

// ItemCategory represents different types of store items
type ItemCategory int

const (
	ItemCategoryUnspecified ItemCategory = iota
	ItemCategoryElectronics
	ItemCategoryClothing
	ItemCategoryBooks
	ItemCategoryHome
	ItemCategorySports
)

// ItemStatus represents the current status of an item
type ItemStatus int

const (
	ItemStatusUnspecified ItemStatus = iota
	ItemStatusActive
	ItemStatusInactive
	ItemStatusOutOfStock
	ItemStatusDiscontinued
)

// Item represents a store item with enhanced fields
type Item struct {
	PK             string       `dynamodbav:"PK"` // Partition key: TENANT#{tenant_id}
	SK             string       `dynamodbav:"SK"` // Sort key: ITEM#{item_id}
	ItemID         string       `dynamodbav:"ItemID"`
	TenantID       int64        `dynamodbav:"TenantID"`
	Name           string       `dynamodbav:"Name"`
	Description    string       `dynamodbav:"Description"`
	Price          float64      `dynamodbav:"Price"`
	Category       ItemCategory `dynamodbav:"Category"`
	Status         ItemStatus   `dynamodbav:"Status"`
	SKU            string       `dynamodbav:"SKU"`
	InventoryCount int32        `dynamodbav:"InventoryCount"`
	Tags           []string     `dynamodbav:"Tags,omitempty"`
	CreatedAt      time.Time    `dynamodbav:"CreatedAt"`
	UpdatedAt      time.Time    `dynamodbav:"UpdatedAt"`
	CreatedBy      string       `dynamodbav:"CreatedBy"`
	UpdatedBy      string       `dynamodbav:"UpdatedBy"`
}

// StoreInterface defines the interface for store operations
type StoreInterface interface {
	CreateItem(ctx context.Context, tenantID int64, name, description string, price float64, category ItemCategory, sku string, inventoryCount int32, tags []string, createdBy string) (Item, error)
	GetItem(ctx context.Context, tenantID int64, itemID string) (Item, error)
	UpdateItem(ctx context.Context, tenantID int64, itemID, name, description string, price float64, category ItemCategory, status ItemStatus, sku string, inventoryCount int32, tags []string, updatedBy string) (Item, error)
	DeleteItem(ctx context.Context, tenantID int64, itemID string) error
	ListItems(ctx context.Context, tenantID int64, category ItemCategory, status ItemStatus, searchQuery string, pageSize int32, pageToken string) ([]Item, string, int32, error)
	UpdateInventory(ctx context.Context, tenantID int64, itemID string, quantityChange int32, reason, updatedBy string) (Item, int32, error)
}

// DynamoStore implements StoreInterface using DynamoDB
type DynamoStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoStore creates a new DynamoDB store instance
func NewDynamoStore(client *dynamodb.Client, tableName string) *DynamoStore {
	return &DynamoStore{
		client:    client,
		tableName: tableName,
	}
}

// CreateItem creates a new store item
func (s *DynamoStore) CreateItem(ctx context.Context, tenantID int64, name, description string, price float64, category ItemCategory, sku string, inventoryCount int32, tags []string, createdBy string) (Item, error) {
	start := time.Now()

	itemID := uuid.New().String()
	now := time.Now()

	item := Item{
		PK:             fmt.Sprintf("TENANT#%d", tenantID),
		SK:             fmt.Sprintf("ITEM#%s", itemID),
		ItemID:         itemID,
		TenantID:       tenantID,
		Name:           name,
		Description:    description,
		Price:          price,
		Category:       category,
		Status:         ItemStatusActive,
		SKU:            sku,
		InventoryCount: inventoryCount,
		Tags:           tags,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      createdBy,
		UpdatedBy:      createdBy,
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		logging.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"item_name": name,
		}).Error("Failed to marshal item")
		return Item{}, fmt.Errorf("failed to marshal item: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})
	if err != nil {
		logging.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"item_id":   itemID,
		}).Error("Failed to put item")
		return Item{}, fmt.Errorf("failed to put item: %w", err)
	}

	logging.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"item_id":   itemID,
		"duration":  time.Since(start),
	}).Info("Item created successfully")

	return item, nil
}

// GetItem retrieves an item by ID
func (s *DynamoStore) GetItem(ctx context.Context, tenantID int64, itemID string) (Item, error) {
	start := time.Now()

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("TENANT#%d", tenantID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", itemID)},
		},
	})
	if err != nil {
		logging.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"item_id":   itemID,
		}).Error("Failed to get item")
		return Item{}, fmt.Errorf("failed to get item: %w", err)
	}

	if result.Item == nil {
		logging.WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"item_id":   itemID,
		}).Warn("Item not found")
		return Item{}, ErrItemNotFound
	}

	var item Item
	err = attributevalue.UnmarshalMap(result.Item, &item)
	if err != nil {
		logging.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"item_id":   itemID,
		}).Error("Failed to unmarshal item")
		return Item{}, fmt.Errorf("failed to unmarshal item: %w", err)
	}

	logging.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"item_id":   itemID,
		"duration":  time.Since(start),
	}).Debug("Item retrieved successfully")

	return item, nil
}

// UpdateItem updates an existing item
func (s *DynamoStore) UpdateItem(ctx context.Context, tenantID int64, itemID, name, description string, price float64, category ItemCategory, status ItemStatus, sku string, inventoryCount int32, tags []string, updatedBy string) (Item, error) {
	start := time.Now()
	now := time.Now()

	// Build update expression
	updateExpr := "SET #name = :name, #desc = :desc, #price = :price, #category = :category, #status = :status, #sku = :sku, #inventory = :inventory, #tags = :tags, #updatedAt = :updatedAt, #updatedBy = :updatedBy"

	exprAttrNames := map[string]string{
		"#name":      "Name",
		"#desc":      "Description",
		"#price":     "Price",
		"#category":  "Category",
		"#status":    "Status",
		"#sku":       "SKU",
		"#inventory": "InventoryCount",
		"#tags":      "Tags",
		"#updatedAt": "UpdatedAt",
		"#updatedBy": "UpdatedBy",
	}

	exprAttrValues := map[string]types.AttributeValue{
		":name":      &types.AttributeValueMemberS{Value: name},
		":desc":      &types.AttributeValueMemberS{Value: description},
		":price":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", price)},
		":category":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", int(category))},
		":status":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", int(status))},
		":sku":       &types.AttributeValueMemberS{Value: sku},
		":inventory": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", inventoryCount)},
		":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		":updatedBy": &types.AttributeValueMemberS{Value: updatedBy},
	}

	// Handle tags
	if len(tags) > 0 {
		tagsList := make([]types.AttributeValue, len(tags))
		for i, tag := range tags {
			tagsList[i] = &types.AttributeValueMemberS{Value: tag}
		}
		exprAttrValues[":tags"] = &types.AttributeValueMemberL{Value: tagsList}
	} else {
		exprAttrValues[":tags"] = &types.AttributeValueMemberL{Value: []types.AttributeValue{}}
	}

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("TENANT#%d", tenantID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", itemID)},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
	})
	if err != nil {
		logging.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"item_id":   itemID,
		}).Error("Failed to update item")
		return Item{}, fmt.Errorf("failed to update item: %w", err)
	}

	logging.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"item_id":   itemID,
		"duration":  time.Since(start),
	}).Info("Item updated successfully")

	// Return the updated item
	return s.GetItem(ctx, tenantID, itemID)
}

// DeleteItem soft-deletes an item by setting status to discontinued
func (s *DynamoStore) DeleteItem(ctx context.Context, tenantID int64, itemID string) error {
	start := time.Now()

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("TENANT#%d", tenantID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", itemID)},
		},
		UpdateExpression: aws.String("SET #status = :status, #updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status":    "Status",
			"#updatedAt": "UpdatedAt",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", int(ItemStatusDiscontinued))},
			":updatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
	})
	if err != nil {
		logging.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"item_id":   itemID,
		}).Error("Failed to delete item")
		return fmt.Errorf("failed to delete item: %w", err)
	}

	logging.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"item_id":   itemID,
		"duration":  time.Since(start),
	}).Info("Item deleted successfully")

	return nil
}

// ListItems lists items with filtering and pagination
func (s *DynamoStore) ListItems(ctx context.Context, tenantID int64, category ItemCategory, status ItemStatus, searchQuery string, pageSize int32, pageToken string) ([]Item, string, int32, error) {
	start := time.Now()

	// This is a simplified implementation - in production you'd want proper GSI for filtering
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: fmt.Sprintf("TENANT#%d", tenantID)},
			":sk_prefix": &types.AttributeValueMemberS{Value: "ITEM#"},
		},
		Limit: aws.Int32(pageSize),
	}

	if pageToken != "" {
		// Simplified pagination - in production, you'd properly encode/decode the last evaluated key
		input.ExclusiveStartKey = map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("TENANT#%d", tenantID)},
			"SK": &types.AttributeValueMemberS{Value: pageToken},
		}
	}

	result, err := s.client.Query(ctx, input)
	if err != nil {
		logging.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
		}).Error("Failed to list items")
		return nil, "", 0, fmt.Errorf("failed to list items: %w", err)
	}

	var items []Item
	for _, item := range result.Items {
		var i Item
		err := attributevalue.UnmarshalMap(item, &i)
		if err != nil {
			logging.WithError(err).Error("Failed to unmarshal item in list")
			continue
		}

		// Apply filters (in production, use GSI for better performance)
		if category != ItemCategoryUnspecified && i.Category != category {
			continue
		}
		if status != ItemStatusUnspecified && i.Status != status {
			continue
		}

		items = append(items, i)
	}

	nextPageToken := ""
	if result.LastEvaluatedKey != nil {
		if sk, ok := result.LastEvaluatedKey["SK"]; ok {
			if skValue, ok := sk.(*types.AttributeValueMemberS); ok {
				nextPageToken = skValue.Value
			}
		}
	}

	logging.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"items_count": len(items),
		"duration":    time.Since(start),
	}).Debug("Items listed successfully")

	return items, nextPageToken, int32(len(items)), nil
}

// UpdateInventory updates the inventory count for an item
func (s *DynamoStore) UpdateInventory(ctx context.Context, tenantID int64, itemID string, quantityChange int32, reason, updatedBy string) (Item, int32, error) {
	start := time.Now()

	// First get the current item to know the previous inventory count
	currentItem, err := s.GetItem(ctx, tenantID, itemID)
	if err != nil {
		return Item{}, 0, err
	}

	previousCount := currentItem.InventoryCount
	newCount := previousCount + quantityChange

	// Ensure inventory doesn't go negative
	if newCount < 0 {
		return Item{}, previousCount, fmt.Errorf("insufficient inventory: current=%d, requested_change=%d", previousCount, quantityChange)
	}

	// Update the inventory count
	now := time.Now()
	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("TENANT#%d", tenantID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", itemID)},
		},
		UpdateExpression: aws.String("SET #inventory = :inventory, #updatedAt = :updatedAt, #updatedBy = :updatedBy"),
		ExpressionAttributeNames: map[string]string{
			"#inventory": "InventoryCount",
			"#updatedAt": "UpdatedAt",
			"#updatedBy": "UpdatedBy",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inventory": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", newCount)},
			":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":updatedBy": &types.AttributeValueMemberS{Value: updatedBy},
		},
	})
	if err != nil {
		logging.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"item_id":   itemID,
		}).Error("Failed to update inventory")
		return Item{}, previousCount, fmt.Errorf("failed to update inventory: %w", err)
	}

	logging.WithFields(logrus.Fields{
		"tenant_id":       tenantID,
		"item_id":         itemID,
		"previous_count":  previousCount,
		"quantity_change": quantityChange,
		"new_count":       newCount,
		"reason":          reason,
		"duration":        time.Since(start),
	}).Info("Inventory updated successfully")

	// Return the updated item
	updatedItem, err := s.GetItem(ctx, tenantID, itemID)
	if err != nil {
		return Item{}, previousCount, err
	}

	return updatedItem, previousCount, nil
}
