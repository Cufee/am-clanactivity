package mongoapi

import (
	"fmt"
	"log"
	"time"

	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/cufee/am-clanactivity/config"
)

// BSON

// TankAverages - Struct for getting tank averages data from DB
type TankAverages struct {
	All struct {
		Battles              float64 `bson:"battles,omitempty"`
		DroppedCapturePoints float64 `bson:"dropped_capture_points,omitempty"`
	} `bson:"all"`
	Special struct {
		Winrate         float64 `bson:"winrate,omitempty"`
		DamageRatio     float64 `bson:"damageRatio,omitempty"`
		Kdr             float64 `bson:"kdr,omitempty"`
		DamagePerBattle float64 `bson:"damagePerBattle,omitempty"`
		KillsPerBattle  float64 `bson:"killsPerBattle,omitempty"`
		HitsPerBattle   float64 `bson:"hitsPerBattle,omitempty"`
		SpotsPerBattle  float64 `bson:"spotsPerBattle,omitempty"`
		Wpm             float64 `bson:"wpm,omitempty"`
		Dpm             float64 `bson:"dpm,omitempty"`
		Kpm             float64 `bson:"kpm,omitempty"`
		HitRate         float64 `bson:"hitRate,omitempty"`
		SurvivalRate    float64 `bson:"survivalRate,omitempty"`
	} `bson:"special"`
	Name   string `bson:"name"`
	Tier   int    `bson:"tier"`
	Nation string `bson:"nation"`
}

// Clan DB record struct
type Clan struct {
	ID         int       `bson:"_id" json:"clan_id"`
	ClanName   string    `bson:"clan_name" json:"clan_name"`
	ClanTag    string    `bson:"clan_tag" json:"clan_tag"`
	MembersIds []int     `bson:"members_ids" json:"members_ids"`
	Realm      string    `bson:"realm" json:"realm"`
	LastUpdate time.Time `bson:"last_update" json:"last_update"`
}

// Player DB record struct
type Player struct {
	ID                int       `bson:"_id" json:"player_id"`
	JoinedAt          int       `bson:"joined_at" json:"joined_at"`
	Nickname          string    `bson:"nickname" json:"nickname"`
	PremiumExpiration int       `bson:"premium_expiration" json:"premium_expiration"`
	AverageRating     int       `bson:"average_rating" json:"average_rating"`
	Battles           int       `bson:"battles" json:"battles"`
	SessionBattles    int       `json:"session_battles"`
	SessionRating     int       `json:"session_rating"`
	LastUpdate        time.Time `bson:"last_update" json:"last_update"`
}

// Collections
var clansCollection *mongo.Collection
var playersCollection *mongo.Collection
var tankAveragesCollection *mongo.Collection
var ctx = context.TODO()

func init() {
	// Conenct to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoURI))
	if err != nil {
		log.Println("Panic in mongoapi/init")
		panic(err)
	}
	// Ping the primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Println("Panic in mongoapi/init")
		panic(err)
	}
	log.Println("Successfully connected and pinged.")

	// Collections
	clansCollection = client.Database("clan_activity").Collection("clans")
	playersCollection = client.Database("clan_activity").Collection("players")
	tankAveragesCollection = client.Database("glossary").Collection("tankaverages")
}

// CLANS

// GetClan - Retrieve clan record from db using bson.M filter
func GetClan(filter interface{}) (Clan, error) {
	var clanData Clan
	err := clansCollection.FindOne(ctx, filter).Decode(&clanData)
	if err != nil {
		return clanData, err
	}
	return clanData, nil
}

// UpdateClan - Update a clan record in a db, with optional upsert
func UpdateClan(clanData Clan, upsert bool) (string, error) {
	// set upsert
	var opts *options.UpdateOptions
	if upsert {
		opts = options.Update().SetUpsert(true)
	} else {
		opts = options.Update().SetUpsert(false)
	}
	// Set LastUpdate
	loc, _ := time.LoadLocation("UTC")
	clanData.LastUpdate = time.Now().In(loc)
	// Update and return result/error
	filter := bson.M{"_id": clanData.ID}
	result, err := clansCollection.UpdateOne(ctx, filter, bson.M{"$set": clanData}, opts)
	if err != nil {
		return "mongoapi/UpdateClan: Error updating clan record.", err
	}
	resultStr := fmt.Sprintf("%+v", result)
	return resultStr, nil
}

// PLAYERS

// GetPlayer - Retrieve player record from db using bson.M filter
func GetPlayer(filter interface{}) (Player, error) {
	var playerData Player
	err := playersCollection.FindOne(ctx, filter).Decode(&playerData)
	if err != nil {
		return playerData, err
	}
	return playerData, nil
}

// UpdatePlayer - Update a player record in a db, with optional upsert
func UpdatePlayer(playerData Player, upsert bool) (string, error) {
	// set upsert
	var opts *options.UpdateOptions
	if upsert {
		opts = options.Update().SetUpsert(true)
	} else {
		opts = options.Update().SetUpsert(false)
	}
	// Set LastUpdate
	loc, _ := time.LoadLocation("UTC")
	playerData.LastUpdate = time.Now().In(loc)
	// Update and return result/error
	filter := bson.M{"_id": playerData.ID}
	result, err := playersCollection.UpdateOne(ctx, filter, bson.M{"$set": playerData}, opts)
	if err != nil {
		return "mongoapi/UpdatePlayer: Error updating player record.", err
	}
	resultStr := fmt.Sprintf("%+v", result)
	return resultStr, nil
}

// TANKAVERAGES

// GetTankAvg - Get averages data for a tank using a bson.M filter
func GetTankAvg(filter interface{}) (TankAverages, error) {
	var tankData TankAverages
	err := tankAveragesCollection.FindOne(ctx, filter).Decode(&tankData)
	if err != nil {
		return tankData, err
	}
	return tankData, nil
}
