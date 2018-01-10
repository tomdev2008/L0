-- 用合约来完成一个数字货币系统
local L0 = require("L0")

-- 合约创建时会被调用一次，之后就不会被调用
function L0Init(args)
    --print("in L0Init")
    L0.PutState("VERSION", 0)
    local tabs={amount=10000000, tags="system"}
    L0.PutState("System", tabs)

    --local cnt = L0.GetState("Value")

    local cnt = L0.PutState("Value", 0)
    --print("=====>>>>> ", cnt)
    --L0.sleep(1000)
    return true
end

-- 每次合约执行都调用
function L0Invoke(func, args)
    --print("in L0Invoke")
    --L0.sleep(5)
    local cnt = L0.GetState("Value")
    cnt = cnt +1
    --L0.sleep(1)
    for count=500, 1, -1 do
        local tmp = string.format("==> %d", count)
        L0.PutState(tmp, count)
    end

    L0.PutState("Value", cnt)
    if (cnt % 1000 ~= 0) then
        print("<<<>>>>>222<<<<>cnt: ", cnt)
    end

    -- print(args)
    if("send" == func) then
        send(args)
    elseif("transfer" == func) then
        transfer(args)
    end

    return true
end

-- 查询
function L0Query(args)
    --print("in L0Query")
    --local tabs = L0.GetState(args[0])
    --local amount = L0.GetState(args[0].."amount")
    --local balances = L0.GetState("System")
    --print("value", tabs["tags"], tabs["amount"], amount, balances["amount"])
    return "L0query ok"
end

function send(args)
    L0.PutState(args[0], tabs)
    L0.PutState(args[0].."tags", args[2])
    L0.PutState(args[0].."amount", args[1])
    -- print("send...", args[0], args[1], args[2])
end

--
--function transfer(receiver, amount)
--    print("do transfer print by lua",receiver,amount)
--    L0.Transfer(receiver, 0, amount)
--end