-- 用合约来完成一个数字货币系统
local L0 = require("L0")

-- 合约创建时会被调用一次，之后就不会被调用
function L0Init(args)
    print("in L0Init")
    L0.PutState("version", 0)
    local tabs={amount=10000000, tags="system"}
    L0.PutState("System", tabs)
    return true
end

-- 每次合约执行都调用
function L0Invoke(func, args)
    print("in L0Invoke")

    if("send" == func) then
        send(args)
    elseif("transfer" == func) then
        transfer(args)
    end

    return true
end

-- 查询
function L0Query(args)
    print("in L0Query")
    return "L0query ok"
end

function send(args)
    local sender = L0.Account().Address
    local balances = L0.GetState("System")
    balances = balances["amount"] - tonumber(args[1])
    local tabs = {amount=tonumber(args[1]), tags=args[2]}
    L0.PutState(args[0], tabs)
    L0.PutState(args[0].."tags", args[2])
    L0.PutState(args[0].."amount", args[1])
    L0.PutState("balances", balances)
end

function transfer(receiver, amount)
    print("do transfer print by lua",receiver,amount)
    L0.Transfer(receiver, 0, amount)
end