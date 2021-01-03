[中文文档](README_zh-cn.md)

## Use mongodb driver to solve distributed transactions


#### Solving distributed transaction scenarios

This project can only solve distributed transactions in multiple tables in the same database in the same replica-set in mongodb.



#### Main principles

The mongodb server does not strictly bind the transaction in which the transaction is in, and the mongodb driver can initiate transactions to achieve this.
Changing this solution will extend the mongodb golang driver code,
It can be used for verification in mongodb driver 1.1.2 release and mongodb 4.0.x-4.2.x replic-set mode

###### mongodb golang driver code extension content

Extension code:
```
 mongo/session_exposer.go
 x/mongo/driver/session/session_ext.go
```
The extension code is mainly the underlying logic, used to activate the transaction and bind the transaction. Does not contain business logic



Related test cases:

```
Transaction/Transaction_test.go

```

Upper-level use logic:

```
Transaction/Transaction.go
```

Encapsulate business logic, realize business-level management interfaces such as opening, committing, and rolling back transactions, and provide an operation interface for the transaction uuid and the cursor id record operation of the statement execution within the transaction.
In actual use, the transaction uuid and in-service statement execution cursor id need to be stored centrally, and all service instances need to be available and used. You can use redis and mysql as central storage


###### Specific implementation

1. Activate a transaction, generate a transaction uuid (mongodb driver provides the generation method), Transaction/Transaction.go: StartTransaction
2. Activate a session through the uuid of the transaction and join a transaction of the mongodb server, Transaction/Transaction.go: ReloadSession
3. Bind the transaction to the session, get the SessionContext, mongo/session_exposer.go:TxnContextWithSession
4. Transaction execution needs to move the cursor, Transaction/Transaction.go: NextTransactionCursor
5. Perform curl operations



#### requires attention

-The TxnNumber in the current code is only for testing and cannot be used in online multi-service scenarios
-Uuid in TxnNumber must use go.mongodb.org/mongo-driver/x/mongo/driver/uuid base64 encoded string value





#### Solution discovery and implementation

participants:
[rentiansheng](https://github.com/rentiansheng)

[wusendong](https://github.com/wusendong)

[breezelxp](https://github.com/breezelxp)


Use project:


[Blue Whale Configuration Platform](https://github.com/Tencent/bk-cmdb)

#### Drainage
[bson decode register](https://github.com/rentiansheng/bson-register)

Bson decoding interface priority uses the actual type as the final object, such as slice, map
In the mongodb golang official driver, when bson uses interface at the top level, it puts slice and map in an object called primitive.D (golang []interface type).
