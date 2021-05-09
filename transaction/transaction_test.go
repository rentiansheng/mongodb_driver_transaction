package Transaction

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var dbName string

func initMongoClient(t *testing.T) {
	disableWriteRetry := false
	maxPoolSize, minPoolSize, timeout := uint64(100), uint64(10), time.Second
	/*
		set os environment variable:

		export mongodb_connect_uri=mongodb://user:password@ip:port/dbname?authMechanism=SCRAM-SHA-1
		export mongodb_rsname=replica set name
		export mongodb_dbname=dbname
	*/
	rsName := os.Getenv("mongodb_rsname")
	cnnectURI := os.Getenv("mongodb_connect_uri")
	dbName = os.Getenv("mongodb_dbname")
	conOpt := options.ClientOptions{
		MaxPoolSize:    &maxPoolSize,
		MinPoolSize:    &minPoolSize,
		ConnectTimeout: &timeout,
		SocketTimeout:  &timeout,
		ReplicaSet:     &rsName,
		RetryWrites:    &disableWriteRetry,
	}

	var err error
	client, err = mongo.NewClient(options.Client().ApplyURI(cnnectURI), &conOpt)
	if nil != err {
		t.Error(err.Error())
		return
	}

	if err := client.Connect(context.TODO()); nil != err {
		t.Error(err.Error())
		return
	}

}

func TestTransaction(t *testing.T) {
	initMongoClient(t)
	ctx := context.TODO()
	db := client.Database(dbName)
	tableName := "test"
	db.Collection(tableName).Drop(ctx)
	db.RunCommand(ctx, map[string]interface{}{"create": tableName})

	table := db.Collection(tableName)
	txnSess, txnUUID, err := StartTransaction(ctx, client)
	if err != nil {
		t.Error(err)
		return
	}
	txnCtx1 := mongo.TxnContextWithSession(ctx, txnSess)

	txnSess2, txnUUID2, err := StartTransaction(ctx, client)
	if err != nil {
		t.Error(err)
		return
	}
	txnCtx2 := mongo.TxnContextWithSession(ctx, txnSess2)

	// 在事务外插入数据
	// Insert data outside the transaction
	rows := []interface{}{map[string]interface{}{"raw": 1}, map[string]interface{}{"raw": 2}}
	if _, err := table.InsertMany(ctx, rows); err != nil {
		t.Error(err)
		return
	}
	_, err = table.InsertOne(txnCtx1, map[string]interface{}{"txn": 1})
	if err != nil {
		t.Error(err)
		return
	}

	// 事务外查询数据， 看是否能看到事务内新加的数据
	// Query data outside the transaction to see if you can see the newly added data in the transaction
	cnt, err := table.CountDocuments(ctx, map[string]interface{}{"txn": 1})
	if err != nil {
		t.Error(err)
		return
	}
	// 检查事务内数据
	// Check the data in the transaction
	if cnt != 0 {
		t.Error("transaction failed, find transaction data")
		return
	}

	filter := map[string]interface{}{"txn": 1}
	doc := map[string]interface{}{"$set": map[string]interface{}{"txn": 2}}
	_, err = table.UpdateOne(txnCtx1, filter, doc)
	if err != nil {
		t.Error(err)
		return
	}

	// 事务外查询数据， 看是否能看到事务内新加的数据
	// Query data outside the transaction to see if you can see the newly added data in the transaction
	cnt, err = table.CountDocuments(ctx, map[string]interface{}{"txn": 2})
	if err != nil {
		t.Error(err)
		return
	}
	if cnt != 0 {
		t.Error("transaction failed, find transaction data")
		return
	}

	// 事务2， 新加数据
	// Transaction 2, new data
	_, err = table.InsertOne(txnCtx2, map[string]interface{}{"txn2": 1})
	if err != nil {
		t.Error(err)
		return
	}

	// 事务外查询数据， 看是否能看到事务2内新加的数据
	// Query data outside the transaction to see if you can see the newly added data in transaction 2
	cnt, err = table.CountDocuments(txnCtx1, map[string]interface{}{"txn2": 1})
	if err != nil {
		t.Error(err)
		return
	}

	// 检查事务1内数据
	// Check the data in transaction 1
	if cnt != 0 {
		t.Error("transaction failed, find transaction 2 data")
		return
	}

	// 提交事务
	// commit transaction
	if err := CommitTransaction(txnCtx1, txnUUID); err != nil {
		t.Error(err)
		return
	}

	if err := AbortTransaction(txnCtx2, txnUUID2); err != nil {
		t.Error(err)
		return
	}

	// 查询事务提交的数据
	// Query the data submitted by the transaction
	cnt, err = table.CountDocuments(ctx, map[string]interface{}{"txn": 2})
	if err != nil {
		t.Error(err)
		return
	}
	if cnt == 0 {
		t.Error("Transaction commit failed")
		return
	}

	// 查询事务回滚的数据
	// Query the data aborted by the transaction
	cnt, err = table.CountDocuments(ctx, map[string]interface{}{"txn2": 1})
	if err != nil {
		t.Error(err)
		return
	}
	if cnt != 0 {
		t.Error("Transaction abort failed")
		return
	}

}

func TestDistributeTransaction(t *testing.T) {
	initMongoClient(t)
	ctx := context.TODO()
	db := client.Database(dbName)
	tableName := "test"
	db.Collection(tableName).Drop(ctx)
	db.RunCommand(ctx, map[string]interface{}{"create": tableName})

	table := db.Collection(tableName)
	txnSessNode1, txnUUID, err := StartTransaction(ctx, client)
	if err != nil {
		t.Error(err)
		return
	}
	txnCtxNode1 := mongo.TxnContextWithSession(ctx, txnSessNode1)

	txnSessNode2, err := ReloadSession(ctx, client, txnUUID)
	if err != nil {
		t.Error(err)
		return
	}
	txnCtxNode2 := mongo.TxnContextWithSession(ctx, txnSessNode2)

	// node1 insert row
	_, err = table.InsertOne(txnCtxNode1, map[string]interface{}{"txn": "node1"})
	if err != nil {
		t.Error(err)
		return
	}

	// node2 count row
	cnt, err := table.CountDocuments(txnCtxNode2, map[string]interface{}{"txn": "node1"})
	if err != nil {
		t.Error(err)
		return
	}
	if cnt == 0 {
		t.Errorf("not found data")
		return
	}

	// node2 insert row
	_, err = table.InsertOne(txnCtxNode2, map[string]interface{}{"txn": "node2"})
	if err != nil {
		t.Error(err)
		return
	}

	// node1  commit transaction
	if err := CommitTransaction(txnCtxNode1, txnUUID); err != nil {
		t.Error(err)
		return
	}

	// Query the data submitted by the transaction
	cnt, err = table.CountDocuments(context.Background(), map[string]interface{}{"txn": "node2"})
	if err != nil {
		t.Error(err)
		return
	}
	if cnt == 0 {
		t.Error("Transaction commit failed")
		return
	}

}
