package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection      *mongo.Collection
	auctionDuration time.Duration
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection:      database.Collection("auctions"),
		auctionDuration: getAuctionDuration(),
	}
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	go ar.closeAuctionWhenExpires(auctionEntity.Id, auctionEntity.Timestamp)

	return nil
}

func (ar *AuctionRepository) closeAuctionWhenExpires(auctionId string, createdAt time.Time) {
	remaining := time.Until(createdAt.Add(ar.auctionDuration))
	if remaining > 0 {
		<-time.After(remaining)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	filter := bson.M{"_id": auctionId, "status": auction_entity.Active}
	update := bson.M{"$set": bson.M{"status": auction_entity.Completed}}

	result, err := ar.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error(fmt.Sprintf("Error trying to close auction %s", auctionId), err)
		return
	}

	if result.ModifiedCount > 0 {
		logger.Info(fmt.Sprintf("Auction %s closed automatically", auctionId))
	}
}

func getAuctionDuration() time.Duration {
	duration, err := time.ParseDuration(os.Getenv("AUCTION_DURATION"))
	if err != nil {
		return time.Minute * 5
	}

	return duration
}
