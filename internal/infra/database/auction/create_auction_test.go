package auction

import (
	"context"
	"os"
	"testing"
	"time"

	"fullcycle-auction_go/internal/entity/auction_entity"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCreateAuctionFechaAutomaticamente(t *testing.T) {
	t.Setenv("AUCTION_DURATION", "3s")

	repository := NewAuctionRepository(conectar(t))

	auctionEntity, errEntity := auction_entity.CreateAuction(
		"Notebook", "eletronicos", "notebook usado em bom estado", auction_entity.Used)
	if errEntity != nil {
		t.Fatalf("erro ao montar o leilao: %v", errEntity.Message)
	}

	ctx := context.Background()
	if err := repository.CreateAuction(ctx, auctionEntity); err != nil {
		t.Fatalf("erro ao criar o leilao: %v", err.Message)
	}

	t.Cleanup(func() {
		repository.Collection.DeleteOne(context.Background(), bson.M{"_id": auctionEntity.Id})
	})

	criado, err := repository.FindAuctionById(ctx, auctionEntity.Id)
	if err != nil {
		t.Fatalf("erro ao buscar o leilao recem criado: %v", err.Message)
	}
	if criado.Status != auction_entity.Active {
		t.Fatalf("esperado status Active na criacao, recebido %v", criado.Status)
	}

	time.Sleep(5 * time.Second)

	fechado, err := repository.FindAuctionById(ctx, auctionEntity.Id)
	if err != nil {
		t.Fatalf("erro ao buscar o leilao apos a expiracao: %v", err.Message)
	}
	if fechado.Status != auction_entity.Completed {
		t.Fatalf("esperado status Completed apos a expiracao, recebido %v", fechado.Status)
	}
}

func conectar(t *testing.T) *mongo.Database {
	t.Helper()

	url := os.Getenv("MONGODB_URL")
	if url == "" {
		url = "mongodb://admin:admin@localhost:27017/auctions?authSource=admin"
	}

	nome := os.Getenv("MONGODB_DB")
	if nome == "" {
		nome = "auctions"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(url))
	if err != nil {
		t.Skipf("mongodb indisponivel em %s: %v", url, err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("mongodb indisponivel em %s: %v", url, err)
	}

	t.Cleanup(func() {
		client.Disconnect(context.Background())
	})

	return client.Database(nome)
}
