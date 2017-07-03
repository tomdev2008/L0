-- 用合约来完成一个投票系统
local L0 = require("L0")

-- 可投票人
local voters = { "张三", "李四", "王五", "赵六", "孙七", "周八", "吴九", "郑十"}
-- 可候选人
local candidates = {"秦皇岛", "大连", "三亚"}

-- --投票人结构
-- voter = {
--     name = "", -- 投票人名字
--     candidateName = "", -- 投票候选人
-- }


-- --候选人结构
-- candidate = {
--     name = "", -- 候选人名字
--     voteCount = 0, -- 候选人中票数
-- }

-- 合约创建时会被调用一次, 完成数据初始化
function L0Init(func, args)
    print("L0Init")
    --初始化参与人
    for k, v in ipairs(voters) 
    do
    local voter = {}
    voter["name"]=v
    L0.PutState("voter:" .. v,voter)
        
    end
    
    --初始化候选项
    for k, v in ipairs(candidates) 
    do
       local candidate = {}
       candidate["name"] = v
       candidate["voteCount"]=0
       L0.PutState("candidate:"..v,candidate)

    end
    
    return true, "ok"
end

-- 每次合约执行都调用
function L0Invoke(func, args)
    print("L0Invoke")
    if func == "vote" then
        local voteName = args[1]
        local candidateName = args[2]
        local voter = L0.GetState("voter:" .. voteName)
        local candidate = L0.GetState("candidate:" .. candidateName)
        print("func ",func,"args ",args[1],args[2],"voter:",voter["name"],"candidate",candidate["name"])        
        if voter == nil or candidate == nil then
            return false, "voter or candidate is nil "
        end
        voter["candidateName"] = candidateName
        L0.PutState("voter:" .. voteName,  voter)
        candidate["voteCount"] = candidate["voteCount"] + 1
        L0.PutState("candidate:" .. candidateName, candidate)
    end
    return true, "ok"
end

-- 每次合约查询都调用
function L0Query(func, args)
    print("L0Query")
    if func == "vote" then
        local voteName = args[1]
        local voter = L0.GetState("voter:" .. voteName)
        return true, "voter: "..voter["name"].." candidateName: "..voter["candidateName"] 
    elseif func == "candidate" then
        local candidateName = args[1]
        local candidate = L0.GetState("candidate:" .. candidateName)
        return true, "candidate: "..candidate["name"].." voteCount: "..candidate["voteCount"]
    elseif func == "max" then
        local victor = nil
        for k, v in ipairs(candidates) do
          local candidate = L0.GetState("candidate:" .. v)
          if victor == nil or candidate["voteCount"] > victor["voteCount"] then 
            victor = candidate
          end
        end
        return true, "victor: "..victor["name"].." voteCount: "..victor["voteCount"]
    end
    return false,"not fund func"..func
end

