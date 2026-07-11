package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ScheduledTask struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	Name        string        `bson:"name"`
	ProjectUUID string        `bson:"project_uuid"`
	Schedule    string        `bson:"schedule"`
	Type        string        `bson:"type"` // "restart"
	OneTime     bool          `bson:"one_time"`
	NextRun     time.Time     `bson:"next_run,omitempty"`
}

var client *mongo.Client
var collection *mongo.Collection
var authorizedUsers *mongo.Collection

func Connect(uri string) error {
	if uri == "" {
		return fmt.Errorf("DB_URL is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var err error
	for i := 0; i < 5; i++ {
		client, err = mongo.Connect(options.Client().ApplyURI(uri))
		if err == nil {
			err = client.Ping(ctx, nil)
			if err == nil {
				break
			}
		}
		log.Printf("Failed to connect to MongoDB, retrying in 2 seconds... (%v)", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return err
	}

	collection = client.Database("coolify_manager").Collection("scheduled_tasks")
	authorizedUsers = client.Database("coolify_manager").Collection("authorized_users")
	log.Println("Connected to MongoDB")
	return nil
}

func AddAuthorizedUser(id int64, role ...string) error {
	if authorizedUsers == nil { return fmt.Errorf("veritabani baglantisi yok") }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel()
	r := "operator"; if len(role)>0 && role[0]!="" { r=role[0] }
	_, err := authorizedUsers.UpdateOne(ctx, bson.M{"telegram_id": id}, bson.M{"$set": bson.M{"telegram_id": id, "role": r, "updated_at": time.Now()}}, options.UpdateOne().SetUpsert(true))
	return err
}

func RemoveAuthorizedUser(id int64) error {
	if authorizedUsers == nil { return fmt.Errorf("veritabani baglantisi yok") }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel()
	_, err := authorizedUsers.DeleteOne(ctx, bson.M{"telegram_id": id}); return err
}

func IsAuthorizedUser(id int64) bool {
	if authorizedUsers == nil { return false }
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel()
	return authorizedUsers.FindOne(ctx, bson.M{"telegram_id": id}).Err() == nil
}

func AuthorizedRole(id int64) string {
	if authorizedUsers == nil { return "" }
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel()
	var row struct{ Role string `bson:"role"` }; if authorizedUsers.FindOne(ctx,bson.M{"telegram_id":id}).Decode(&row)!=nil{return ""}; if row.Role==""{return "operator"};return row.Role
}

type AuthorizedUser struct { TelegramID int64 `bson:"telegram_id" json:"telegram_id"`; Role string `bson:"role" json:"role"` }
func GetAuthorizedUserRecords() ([]AuthorizedUser,error){if authorizedUsers==nil{return nil,fmt.Errorf("veritabani baglantisi kurulamadi")};ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel();cur,err:=authorizedUsers.Find(ctx,bson.M{});if err!=nil{return nil,err};defer cur.Close(ctx);var rows []AuthorizedUser;err=cur.All(ctx,&rows);return rows,err}
func GetAuthorizedUsers() ([]int64, error) {
	if authorizedUsers == nil { return nil, fmt.Errorf("veritabani baglantisi yok") }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel()
	cur, err := authorizedUsers.Find(ctx, bson.M{}); if err != nil { return nil, err }; defer cur.Close(ctx)
	var rows []struct { TelegramID int64 `bson:"telegram_id"` }; if err=cur.All(ctx,&rows); err!=nil{return nil,err}
	ids:=make([]int64,0,len(rows)); for _,row:=range rows{ids=append(ids,row.TelegramID)}; return ids,nil
}

func AddTask(task ScheduledTask) error {
	if collection == nil { return fmt.Errorf("veritabani baglantisi yok") }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := collection.InsertOne(ctx, task)
	return err
}

func GetTasks() ([]ScheduledTask, error) {
	if collection == nil { return nil, fmt.Errorf("veritabani baglantisi yok") }
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []ScheduledTask
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func DeleteTask(id string) error {
	if collection == nil { return fmt.Errorf("veritabani baglantisi yok") }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = collection.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

func GetDueOneTimeTasks() ([]ScheduledTask, error) {
	if collection == nil { return nil, fmt.Errorf("veritabani baglantisi yok") }
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"one_time": true,
		"next_run": bson.M{"$lte": time.Now()},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []ScheduledTask
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func RemoveOneTimeTask(id bson.ObjectID) error {
	if collection == nil { return fmt.Errorf("veritabani baglantisi yok") }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func UpdateTaskNextRun(id bson.ObjectID, nextRun time.Time) error {
	if collection == nil { return fmt.Errorf("veritabani baglantisi yok") }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.UpdateByID(ctx, id, bson.M{"$set": bson.M{"next_run": nextRun}})
	return err
}
