/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package Transaction

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
)

var (
	client *mongo.Client
)

// CommitTransaction 提交事务
func CommitTransaction(ctx context.Context, txnUUID string) error {

	reloadSession, _, err := reloadSession(ctx, client, txnUUID)
	if err != nil {
		return err
	}

	// we commit the transaction with the session id
	err = reloadSession.CommitTransaction(ctx)
	if err != nil {
		return fmt.Errorf("commit transaction: %s failed, err: %v", txnUUID, err)
	}

	return nil
}

// AbortTransaction 取消事务
func AbortTransaction(ctx context.Context, txnUUID string) error {
	reloadSession, _, err := reloadSession(ctx, client, txnUUID)
	if err != nil {
		return err
	}

	// we abort the transaction with the session id
	err = reloadSession.AbortTransaction(ctx)
	if err != nil {
		return fmt.Errorf("abort transaction: %s failed, err: %v", txnUUID, err)
	}

	return nil
}

func StartTransaction(ctx context.Context, cli *mongo.Client) (mongo.Session, string, error) {
	return reloadSession(ctx, cli, "")
}

func ReloadSession(ctx context.Context, cli *mongo.Client, txnUUID string) (mongo.Session, error) {
	sess, _, err := reloadSession(ctx, cli, txnUUID)
	return sess, err
}

func reloadSession(ctx context.Context, cli *mongo.Client, txnUUID string) (mongo.Session, string, error) {
	// create a session client.
	sess, err := cli.StartSession()
	if err != nil {
		return nil, txnUUID, fmt.Errorf("start session failed, err: %v", err)
	}

	// only for changing the transaction status
	err = sess.StartTransaction()
	if err != nil {
		return nil, txnUUID, fmt.Errorf("start transaction %s failed: %v", txnUUID, err)
	}

	var txnNumber int64
	if txnUUID == "" {
		mUUID, err := uuid.New()
		if err != nil {
			return nil, txnUUID, fmt.Errorf("generate txn number failed, err: %v", err)
		}
		txnUUID = base64.StdEncoding.EncodeToString(mUUID[:])
		txnNumber = 1
	} else {
		txnNumber = 2

	}

	// reset the session info with the session id.
	info := &mongo.TxnSession{
		TxnNubmer: txnNumber,
		SessionID: txnUUID,
	}

	err = mongo.TnxReloadSession(sess, info)
	if err != nil {
		return nil, txnUUID, fmt.Errorf("reload transaction: %s failed, err: %v", txnUUID, err)
	}

	return sess, txnUUID, nil
}
