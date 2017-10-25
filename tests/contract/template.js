// 用合约来完成 
// 合约创建时会被调用一次, 完成数据初始化
function L0Init(args) {
     console.log("call L0Init");
    
    return true;
}

// 每次合约执行都调用
function L0Invoke(func, args) {
    console.log("call L0Invoke");
    return true;
}


// 每次合约查询都调用
function L0Query(args) {
    return "query ok"
}