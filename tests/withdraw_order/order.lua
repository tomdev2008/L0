-- 用合约来完成一个订单清算(撮合)系统
local L0 = require("L0")
-- 订单清算(撮合)
local CName = "order" 
local Scale = 1
string.split = function(s, p)
    local rt= {}
    string.gsub(s, '[^'..p..']+', function(w) table.insert(rt, w) end )
    return rt
end
table.size = function(tb) 
    local cnt = 0
    for k, v in pairs(tb) do 
        cnt = cnt + 1
    end
    return cnt
end
-- 合约创建时会被调用一次，之后就不会被调用
-- 设置系统账户地址 & 手续费账户地址
function L0Init(args)
    -- info
    local str = ""
    for k, v in pairs(args) do 
        str = str .. v .. ","
    end
    print("INFO:" .. CName .. " L0Init(" .. string.sub(str, 0, -2) .. ")")

    -- validate
    if(table.size(args) ~= 2)
    then
        print("ERR :" .. CName ..  " L0Init --- wrong args number", table.size(args))
        return false
    end
    
    -- execute
    L0.PutState("version", 0)
    print("INFO:" .. CName ..  " L0Init --- system account " .. args[0])
    L0.PutState("account_system", args[0])
    print("INFO:" .. CName ..  " L0Init --- fee account " .. args[1])
    L0.PutState("account_fee", args[1])
    return true
end

-- 每次合约执行都调用
-- 用户账户发起订单 launch & 用户账户撤销订单 cancel & 
-- 系统账户冻结转清算（撮合） matching & 系统账户清算完成（撮合）成功 matched & 系统账户清算（撮合）手续费 feecharge
-- 系统账户撤销订单 syscancel
function L0Invoke(func, args)
    -- info
    local str = ""
    for k, v in pairs(args) do 
        str = str .. v .. ","
    end
    print("INFO:" .. CName ..  " L0Invoke(" .. func .. "," .. string.sub(str, 0, -2) .. ")")

    -- execute
    if("launch" == func) then
        return launch(args)
    elseif("cancel" == func) then
        return cancel(args)
    elseif("matching" == func) then
        return matching(args)
    elseif("matched" == func) then
        return matched(args)
    elseif("feecharge" == func) then
        return feecharge(args)
    elseif("syscancel" == func) then
        return syscancel(args)
    else
        print("ERR :" .. CName ..  " L0Invoke --- function not support", func)
        return false
    end
    return true
end

-- 查询
function L0Query(args)
    -- print info
    local str = ""
    for k, v in pairs(args) do 
        str = str .. v .. ","
    end
    print("INFO:" .. CName ..  " L0Query(" .. string.sub(str, 0, -2) .. ")")
    -- validate
    if(table.size(args) ~= 2)
    then
        print("ERR :" .. CName ..  " L0Query --- wrong args number", table.size(args))
        return false
    end
    -- execute
    if (args[0] == "order") then
        local orderID = "order_"..args[1]
        local orderInfo = L0.GetState(orderID)
        if (not orderInfo)
        then
            return "order " .. args[1] .. " not found " 
        end
        local tb = string.split(orderInfo, "&")
        local amount = tonumber(tb[3])/Scale
        return "order " .. args[1] .. " addr:" .. tb[1] .. " , asset:" .. tb[2] .. " , amount:" .. amount 
    elseif (args[0] == "matching") then
        local matchID = "match_"..args[1]
        local matchInfo_buy = L0.GetState(matchID.."_buy")
        local matchInfo_sell = L0.GetState(matchID.."_sell")
        if (not matchInfo_buy and not matchInfo_sell)
        then
            return "matching " .. args[1] .. " not found "
        end
        local tb_buy = string.split(matchInfo_buy, "&")
        local amount_buy = tonumber(tb_buy[3])/Scale
        if not matchInfo_sell then
            return args[1] .. " addr:" .. tb_buy[1] .. " , asset:" .. tb_buy[2] .. " , amount:" ..  amount_buy
        end
        local tb_sell = string.split(matchInfo_sell, "&")
        local amount_sell = tonumber(tb_sell[3])/Scale
        return "matching " .. args[1] .. " addr:" .. tb_buy[1] .. " , asset:" .. tb_buy[2] .. " , amount:" .. amount_buy .. " addr:" .. tb_sell[1] .. " , asset:" .. tb_sell[2] .. " , amount:" .. amount_sell
    else 
        return args[0] .. " not support "
    end
end

--  用户账户发起订单, 发送方转账到合约账户，保存订单ID（冻结金额）
--  参数：订单ID
--  前置条件: 订单ID不存在
function launch(args) 
    -- validate
    if(table.size(args) ~= 1)
    then
        print("ERR :" .. CName ..  " launch --- wrong args number", table.size(args))
        return false
    end

    -- execute 
    local orderID = "order_"..args[0]
    ----[[
    if (L0.GetState(orderID))
    then
        print("ERR :" .. CName ..  " launch --- orderID alreay exist", args[0])
        return false
    end
    local txInfo = L0.TxInfo()
    local sender = txInfo["Sender"]
    local assetID = txInfo["AssetID"]
    local amount = txInfo["Amount"]
    if (type(sender) ~= "string")
    then
        print("ERR :" .. CName ..  " launch --- wrong sender", sender)
        return false
    end
    if (type(assetID) ~= "number" or assetID < 0)
    then
        print("ERR :" .. CName ..  " launch --- wrong assetID", assetID)
        return false
    end
    if (type(amount) ~= "number" or amount <= 0)
    then
        print("ERR :" .. CName ..  " launch --- wrong amount", amount)
        return false
    end
    L0.PutState(orderID, sender.."&"..assetID.."&"..amount)
    print("INFO:" .. CName ..  " launch ---", orderID, sender, assetID, amount)
    --]]--
    return true
end

--  用户账户撤销订单, 合约账户转账到发送方，删除订单ID
--  参数： 订单ID， 撤销金额
--  前置条件：订单ID存在、发送方正确、金额足够
function cancel(args)
    -- validate
    if(table.size(args) ~= 2)
    then
        print("ERR :" .. CName ..  " cancel --- wrong args number", table.size(args))
        return false
    end
    -- execute
    local orderID = "order_"..args[0]
    local tamount = tonumber(args[1])
    if (not tamount or tamount < 0) 
    then
        print("ERR :" .. CName ..  " cancel --- wrong amount", args[1])
        return false
    end
    tamount = tamount * Scale
    ----[[
    local orderInfo = L0.GetState(orderID)
    if (not orderInfo) 
    then
        print("ERR :" .. CName ..  " cancel --- id not exist", args[0])
        return false
    end
    local txInfo = L0.TxInfo()
    local sender = txInfo["Sender"]
    local amount = txInfo["Amount"]
    if (type(amount) ~= "number" or amount > 0)
    then
        print("ERR :" .. CName ..  " cancel --- wrong tx amount", amount)
        return false
    end
    local tb = string.split(orderInfo, "&")
    local receiver = tb[1]
    local assetID = tonumber(tb[2])
    local amount = tonumber(tb[3])
    if (receiver ~= sender) 
    then
        print("ERR :" .. CName ..  " cancel --- wrong sender", sender, receiver)
        return false
    end
    -- to do balance check
    if (amount < tamount)
    then
        print("ERR :" .. CName ..  " cancel --- balance not enough", amount, tamount)
        return false
    end
    L0.Transfer(receiver, assetID,tamount)
    local b = amount - tamount
    if (b == 0) then
        L0.DelState(orderID)
    else
        L0.PutState(orderID, receiver.."&"..assetID.."&"..b)
    end
    print("INFO:" .. CName ..  " cancel ---", orderID, receiver, assetID, amount, tamount, b)
    --]]--
    return true
end

-- 系统账户冻结转清算（撮合中）
-- 参数：撮合ID、订单ID、撮合金额
-- 前置条件：发送方为系统账户、撮合ID不存在、订单ID已经存在、 订单ID足够金额
function matching(args)
    -- validate
    if(table.size(args) ~= 3)
    then
        print("ERR :" .. "matching --- wrong args number", table.size(args))
        return false
    end
    -- execute
    local matchID = "match_"..args[0]
    local orderID = "order_"..args[1]
    local tamount = tonumber(args[2]) 
    if (not tamount or tamount <0) 
    then
        print("ERR :" .. CName ..  " launch --- wrong amount", args[2])
        return false
    end
    tamount = tamount * Scale
    ----[[
    local system = L0.GetState("account_system")
    local txInfo = L0.TxInfo()
    local sender = txInfo["Sender"]
    if (system ~= sender) 
    then
        print("ERR :" .. CName ..  " matching --- wrong sender", sender, system)
        return false
    end
    local amount = txInfo["Amount"]
    if (type(amount) ~= "number" or amount > 0)
    then
        print("ERR :" .. CName ..  " matching --- wrong tx amount", amount)
        return false
    end
    local matchInfo_buy = L0.GetState(matchID.."_buy")
    local matchInfo_sell = L0.GetState(matchID.."_sell")
    if (matchInfo_buy and matchInfo_sell) 
    then
        print("ERR :" .. CName ..  " matching --- matchID alreay exist", args[0])
        return false
    end
    local orderInfo = L0.GetState(orderID)
    if (not orderInfo) 
    then
        print("ERR :" .. CName ..  " matching --- orderID not exist", args[1])
        return false
    end
    local tb = string.split(orderInfo, "&")
    local receiver = tb[1]
    local assetID = tonumber(tb[2])
    local amount = tonumber(tb[3])
    if (amount < tamount) 
    then
        print("ERR :" .. CName ..  " matching --- balance is not enough", amount, tamount)
        return false
    end

    local b = amount - tamount
    if (b == 0) then
        L0.DelState(orderID)
    else
        L0.PutState(orderID, receiver.."&"..assetID.."&"..b)
    end

    if (not matchInfo_buy)
    then
        L0.PutState(matchID.."_buy",  receiver.."&"..assetID.."&"..tamount)
    else
        L0.PutState(matchID.."_sell",  receiver.."&"..assetID.."&"..tamount)
    end
    print("INFO:" .. CName ..  " matching ---", matchID, orderID, amount, tamount, b)
    --]]--
    return true
end

-- 系统账户清算完成（撮合完成）
-- 参数： 撮合ID
-- 前置条件：发送方为系统账户、撮合ID已经存在
function matched(args)
    -- validate
    if(table.size(args) ~= 1)
    then
        print("ERR :" .. "matched --- wrong args number", table.size(args))
        return false
    end
    -- execute
    local matchID = "match_"..args[0]
    ----[[
    local system = L0.GetState("account_system")
    local txInfo = L0.TxInfo()
    local sender = txInfo["Sender"]
    if (system ~= sender) 
    then
        print("ERR :" .. CName ..  " matched --- wrong sender", sender, system)
        return false
    end
    local amount = txInfo["Amount"]
    if (type(amount) ~= "number" or amount > 0)
    then
        print("ERR :" .. CName ..  " matched --- wrong tx amount", amount)
        return false
    end
    local matchInfo_buy = L0.GetState(matchID.."_buy")
    local matchInfo_sell = L0.GetState(matchID.."_sell")
    if (not matchInfo_buy or not matchInfo_sell) 
    then
        print("ERR :" .. CName ..  " matched --- matchID not exist", args[0])
        return false
    end
    
    local tb_buy = string.split(matchInfo_buy, "&")
    local receiver_buy = tb_buy[1]
    local assetID_buy = tonumber(tb_buy[2])
    local amount_buy = tonumber(tb_buy[3])

    local tb_sell = string.split(matchInfo_sell, "&")
    local receiver_sell = tb_sell[1]
    local assetID_sell = tonumber(tb_sell[2])
    local amount_sell = tonumber(tb_sell[3])
    -- to do balance check
    L0.Transfer(receiver_sell, assetID_buy, amount_buy)
    L0.Transfer(receiver_buy, assetID_sell, amount_sell)
    L0.DelState(matchID.."_buy")
    L0.DelState(matchID.."_sell")
    --]]--
    print("INFO:" .. CName ..  " matched ---", matchID, receiver_buy, assetID_buy, amount_buy, receiver_sell, assetID_sell, amount_sell)
    return true
end

-- 系统账户清算手续费
-- 参数：撮合ID，订单ID、手续费金额
-- 前置条件：发送方为系统账户、撮合ID不存在、订单ID已经存在、 订单ID足够金额
function feecharge(args)
    -- validate
    if(table.size(args) ~= 3)
    then
        print("ERR :" .. "feecharge --- wrong args number", table.size(args))
        return false
    end
    -- execute
    local matchID = "match_"..args[0]
    local orderID = "order_"..args[1]
    local feeamount = tonumber(args[2]) 
    if (not feeamount or feeamount <0) 
    then
        print("ERR :" .. CName ..  " feecharge --- wrong fee amount", args[2])
        return false
    end
    feeamount = feeamount * Scale
    ----[[
    local system = L0.GetState("account_system")
    local txInfo = L0.TxInfo()
    local sender = txInfo["Sender"]
    if (system ~= sender) 
    then
        print("ERR :" .. CName ..  " feecharge --- wrong sender", sender, system)
        return false
    end
    local amount = txInfo["Amount"]
    if (type(amount) ~= "number" or amount > 0)
    then
        print("ERR :" .. CName ..  " feecharge --- wrong tx amount", amount)
        return false
    end

    local orderInfo = L0.GetState(orderID)
    if (not orderInfo) 
    then
        print("ERR :" .. CName ..  " feecharge --- orderID not exist", args[1])
        return false
    end
    local tb = string.split(orderInfo, "&")
    local receiver = tb[1]
    local assetID = tonumber(tb[2])
    local amount = tonumber(tb[3])
    if (amount < feeamount) 
    then
        print("ERR :" .. CName ..  " matching --- balance is not enough", feeAmount, amount)
        return false
    end

    local b = amount - feeamount
    if (b == 0) then
        L0.DelState(orderID)
    else
        L0.PutState(orderID, receiver.."&"..assetID.."&"..b)
    end
    local fee = L0.GetState("account_fee")
    L0.Transfer(fee, assetID, feeamount)
    print("INFO:" .. CName ..  " feecharge ---", matchID, orderID, amount, feeamount, b)
    --]]--
    return true
end

--  系统账户撤销订单, 合约账户转账到发送方，删除订单ID
--  参数： 订单ID， 撤销金额
--  前置条件：订单ID存在、发送方正确、金额足够
function syscancel(args)
    -- validate
    if(table.size(args) ~= 2)
    then
        print("ERR :" .. CName ..  " syscancel --- wrong args number", table.size(args))
        return false
    end
    -- execute
    local orderID = "order_"..args[0]
    local tamount = tonumber(args[1])
    if (not tamount or tamount < 0) 
    then
        print("ERR :" .. CName ..  " syscancel --- wrong amount", args[1])
        return false
    end
    tamount = tamount * Scale
    ----[[
    orderInfo = L0.GetState(orderID)
    if (not orderInfo) 
    then
        print("ERR :" .. CName ..  " syscancel --- id not exist", args[0])
        return false
    end
    local system = L0.GetState("account_system")
    local txInfo = L0.TxInfo()
    local sender = txInfo["Sender"]
    if (system ~= sender) 
    then
        print("ERR :" .. CName ..  " syscancel --- wrong sender", system, sender)
        return false
    end
    local amount = txInfo["Amount"]
    if (type(amount) ~= "number" or amount > 0)
    then
        print("ERR :" .. CName ..  " syscancel --- wrong tx amount", amount)
        return false
    end

    local tb = string.split(orderInfo, "&")
    local receiver = tb[1]
    local assetID = tonumber(tb[2])
    local amount = tonumber(tb[3])
    
    -- to do balance check
    if (amount < tamount)
    then
        print("ERR :" .. CName ..  " syscancel --- balance not enough", amount, tamount)
        return false
    end
    L0.Transfer(receiver, assetID,tamount)

    local b = amount - tamount
    if (b == 0) then
        L0.DelState(orderID)
    else
        L0.PutState(orderID, receiver.."&"..assetID.."&"..b)
    end
    print("INFO:" .. CName ..  " syscancel ---", orderID, receiver, assetID, amount, tamount, b)
    --]]--
    return true
end