
--[[
部署即生效的安全合约。

安全合约对交易做如下验证：
- 验证转出地址和转出账户是否对应
- 验证转入地址和转入账户是否对应
- 验证转出地址是否可转出
- 验证转入地址是否可转入
- 验证转出金额是否超出单笔转账限额
- 验证转出金额是否超出单日转出限额
--]]

local L0 = require("L0")

function L0Init(args)
    return true
end

function L0Invoke(funcName, args)
    if funcName ~= "verify" then
        return false
    end

    -- sender
    local sender = L0.Account().Sender
    local senderAccount = L0.GetGlobalState("account." .. sender)
    if not(senderAccount) then
        return false
    end

    -- receiver
    local receiver = L0.Account().Recipient
    local receiverAccount = L0.GetGlobalState("account." .. receiver)
    if not(receiverAccount) then
        return false
    end

    -- amount
    local amount = L0.Account().Amount

    local singleAmountLimit = tonumber(L0.GetGlobalState("singleTransactionLimit")) or 0
    if singleAmountLimit > 0 and amount > singleAmountLimit then
        return false
    end

    local dailyAmountLimit = tonumber(L0.GetGlobalState("dailyTransactionLimit")) or 0
    if dailyAmountLimit > 0 and amount > dailyAmountLimit then
        return false
    end

    return true
end

function L0Query(args)
    return ""
end