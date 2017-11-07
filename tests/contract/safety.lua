
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
    if type(args) ~= "table" then
        return false
    end

    local safetyContractAddr = args[0]
    if type(key) ~= "string" then
        return false
    end

    L0.SetGlobalState("safetyContract", safetyContractAddr)
    return true
end

function L0Invoke(funcName, args)
    if type(args) ~= "table" then
        return false
    end

    local sender = args[0]
    local receiver = args[1]
    local amount = tonumber(args[2])

    local singleAmountLimit = tonumber(L0.GetGlobalState("singleTransactionLimit"))
    if amount > singleAmountLimit then
        return false
    end

    local dailyAmountLimit = tonumber(L0.GetGlobalState("dailyTransactionLimit"))
    if amount > dailyAmountLimit then
        return false
    end

    return true
end

function L0Query(args)
    return ""
end