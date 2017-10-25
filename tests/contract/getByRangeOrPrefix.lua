-- 用合约来完成 ***
local L0 = require("L0")

-- 合约创建时会被调用一次, 完成数据初始化
function L0Init(args)
     print("L0Init")
     L0.PutState("key_1","value_1")
     L0.PutState("key_11","value_11")
     L0.PutState("key_12","value_12")
     L0.PutState("key_13","value_13")
     L0.PutState("key_14","value_14")     
     L0.PutState("key_2","value_2")
     L0.PutState("key_3","value_3")
     L0.PutState("key_4","value_4")
    return true
end

-- 每次合约执行都调用
function L0Invoke(func, args)
    print("L0Invoke")
    local values1 = L0.GetByPrefix("key_1")
    for k, v in pairs(values1) do
    print("L0Invoke getByPrefix",k,v)
    end

    local values2 = L0.GetByRange("key_1","key_3")
    for k, v in pairs(values2) do
    print("L0Invoke getByPrefix",k,v)
    end
    return true,"ok"
end

-- 每次合约查询都调用
function L0Query(args)

    return true,"query detail"
end