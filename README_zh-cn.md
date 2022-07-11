##  使用mongodb driver 解决分布式事务


#### 解决分布式事务的场景

本项目只能解决在mongodb 同一个replica-set 中同一个database中多个表中的分布式事务。



#### 主要原理

mongodb server对事务所在的session没有严格绑定和 mongodb driver 可以发起事务的规则来实现。 
改该解决方案会对mongodb golang driver 代码扩展，
在mongodb driver 1.1.2 release 与mongodb 4.0.x-4.2.x replica-set 模式验证可以使用

###### mongodb golang driver 代码扩展内容

扩展代码：
```
 mongo/session_exposer.go
 x/mongo/driver/session/session_ext.go
```
扩展代码主要是底层逻辑，用来激活事务， 绑定事务。 不包含业务逻辑



相关测试用例：

```
Transaction/Transaction_test.go

```

上层使用逻辑：

```
Transaction/Transaction.go
```

封装业务逻辑，实现业务层面需要开启，提交，回滚事务等管理接口， 并且提供一个关于事务uuid 与 事务内语句执行游标id记录操作接口。

现在直接将mongodb 真是的事务id暴露出来，在不同服务节点之间传递可能存在安全风险。


###### 具体的实现

1.  激活一个事务， 生成一个事务uuid（mongodb driver 提供生成方法）, Transaction/Transaction.go: StartTransaction
2. 通过事务的uuid， 激活一个session， 加入mongodb server 的一个事务中，Transaction/Transaction.go: ReloadSession
3. 将事务与session绑定， 获取SessionContext， mongo/session_exposer.go:TxnContextWithSession
4. 执行curl 操作






#### 落地

使用项目：


[蓝鲸配置平台](https://github.com/Tencent/bk-cmdb)

#### 引流
[bson decode register](https://github.com/rentiansheng/bson-register)

bson解码interface优先级使用实际类型最为最终对象， 比如：slice，map
mongodb golang 官方driver中bson在顶层使用interface的时候，会将slice, map 放到一个叫primitive.D的对象中（golang []interface类型）。
