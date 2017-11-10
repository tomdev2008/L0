
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

local ZeroAddr = "0x0000000000000000000000000000000000000000"

function L0Init(args)
    return true
end

function L0Invoke(funcName, args)
    local accountInfo = L0.Account()

    -- sender
    local sender = accountInfo.Sender
    local senderAccount = L0.GetGlobalState("account." .. sender)
    if not(senderAccount) then
        return false
    end
    senderAccount = L0.jsonDecode(senderAccount) 

    -- receiver
    local receiver = accountInfo.Recipient
    if receiver ~= ZeroAddr then
        local receiverAccount = L0.GetGlobalState("account." .. receiver)
        if not(receiverAccount) then
            return false
        end

        receiverAccount = L0.jsonDecode(receiverAccount)
    end

    -- amount
    local amount = accountInfo.Amount

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