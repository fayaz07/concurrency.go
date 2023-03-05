package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var counter = -1
var cMutex Counter = Counter{Count: -1, Locker: sync.RWMutex{}}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Auth struct {
	Id       primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	UserId   uint64             `json:"uId" bson:"uId"`
	Email    string             `json:"email" bson:"email"`
	Password string             `json:"password" bson:"password"`
}

func setupRouter() *gin.Engine {
	db := connectToDatabase()
	usersCollection := db.Collection("users")

	cctx, ccancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer ccancel()

	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "uId", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("unique_fields_auth"),
		},
	}
	opts := options.CreateIndexes().SetMaxTime(2 * time.Second)
	_, err := usersCollection.Indexes().CreateMany(cctx, models, opts)
	if err != nil {
		fmt.Println("index creation failed")
	}

	usersMutexCollection := db.Collection("users_mutex")
	_, err = usersMutexCollection.Indexes().CreateMany(cctx, models, opts)
	if err != nil {
		fmt.Println("index creation failed")
	}

	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.POST("/register/raw", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
		defer cancel()
		var body RegisterRequest
		c.BindJSON(&body)

		if counter == -1 {
			// fetch documents count
			count, err := usersCollection.CountDocuments(ctx, bson.M{})
			if err != nil {
				panic(err)
			}
			counter = int(count)
		}
		counter = counter + 1
		usersCollection.InsertOne(ctx, Auth{Email: body.Email, Password: body.Password, UserId: uint64(counter)})
		c.JSON(http.StatusOK, gin.H{})
	})

	r.POST("/register/mutex", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
		defer cancel()
		var body RegisterRequest
		c.BindJSON(&body)

		cMutex.Locker.Lock()
		if cMutex.Count == -1 {
			// fetch documents count
			count, err := usersMutexCollection.CountDocuments(ctx, bson.M{})
			if err != nil {
				panic(err)
			}
			cMutex.Count = count
		}
		cMutex.Count = cMutex.Count + 1
		usersMutexCollection.InsertOne(ctx, Auth{Email: body.Email, Password: body.Password, UserId: uint64(cMutex.Count)})
		cMutex.Locker.Unlock()
		c.JSON(http.StatusOK, gin.H{})
	})

	return r
}

func main() {
	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run("localhost:7070")
}
