
--[[
global 合约。
--]]

local L0 = require("L0")

function L0Init(args)
    return true
end

function L0Invoke(funcName, args)
    if type(args) ~= "table" then
        return false
    end

    local key = args[0]
    if type(key) ~= "string" then
        return false
    end

    if funcName == "SetGlobalState" then
        local value = args[1]
        if not(value) then
            return false
        end
        L0.SetGlobalState(key, value)
        return true
    elseif funcName == "DelGlobalState" then
        L0.DelGlobalState(key)
        return true
    end
    return false
end

function L0Query(args)
    if type(args) ~= "table" then
        return ""
    end

    local key = args[0]
    if type(key) ~= "string" then
        return ""
    end

    return L0.GetGlobalState(key)
end
