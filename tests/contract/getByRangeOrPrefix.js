// 用合约来完成 
// 合约创建时会被调用一次, 完成数据初始化
function L0Init(args) {
     console.log("call L0Init");
     L0.PutState("key_1", "value_1");
     L0.PutState("key_2", "value_2");
     L0.PutState("key_3", "value_3");
     L0.PutState("key_4", "value_4");
     L0.PutState("key_11", "value_11");
     L0.PutState("key_12", "value_12");
     L0.PutState("key_13", "value_13");
    return true;
}

// 每次合约执行都调用
function L0Invoke(func, args) {
    console.log("call L0Invoke");
    var values1 = L0.GetByPrefix("key_1");
   

    for(var i in values1){//用javascript的for/in循环遍历对象的属性
        console.log("key:",i," value:",values1[i]);
    } 

    console.log("----------------");
    
    var values2 = L0.GetByRange("key_1","key_3");
    
    for(var i in values2){//用javascript的for/in循环遍历对象的属性 
        console.log("key:",i," value:",values2[i]);
    } 
    
    return true;
}


// 每次合约查询都调用
function L0Query(args) {
    return "query ok"
}