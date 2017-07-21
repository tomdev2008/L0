-- 投票系统
local L0 = require("L0")

-- 合约创建时会被调用一次，完成数据的初始化
function L0Init(args)
	-- init proposal
	local index = 1
	local proposal = {}
	while(args[index]) 
	do
		proposal[index]	= {["name"] = args[index], ["voteCount"] = 0}
		index = index + 1 
	end
	--local s = serialize(proposal)
	L0.PutState("proposal", proposal)

	-- set chairperson 
	local chairperson = L0.Account().Sender
	L0.PutState("chairperson", chairperson)

	return true
end

-- 每次合约执行都调用
function L0Invoke(func, args)
	if func == "giveVoteTo" then
		local voterAddr = args[0]
		giveVoteTo(voterAddr)
	end
	if func == "vote" then
		local proposalIndex = args[0]
		vote(proposalIndex)
	end
	return true
end
-- 授权
function giveVoteTo(voterAddr)
	local sender = L0.Account().Sender
	local chairperson = L0.GetState("chairperson") 

	if(sender ~= chairperson) 
	then
		print("No access")
		return false
	end

	-- get voters
	local voters = L0.GetState("voters") 
	if voters == nil  -- not exist 
	then
		voters = {}	
	else
		if(voters[voterAddr] ~= nil) 
		then
			-- print(voters[voterAddr]['givenRightTime'])
			print("addr has been given right")
			return false
		end
	end

	--restore
	voters[voterAddr] = {}
	voters[voterAddr]["voteTo"] = -1 

	L0.PutState("voters", voters)
end

-- 投票
function vote(proposalIndex)
	local sender = L0.Account().Sender 
	-- get voters
	local voters = L0.GetState("voters")
	if voters == nil
	then 
		print("No addrs have not been given right")
		return false
	end	

	-- check if given right
	print("sender    "..sender)
	if voters[sender] == nil 
	then
		print("This addr has not been given right")
		return false
	end	
	
	-- check if voted
	if voters[sender]["voteTo"] >= 0
	then
		print("This addr has voted")
		return false
	end
	-- check proposal index
	local proposal = L0.GetState("proposal")

	proposalIndex = tonumber(proposalIndex)
	if(proposal[proposalIndex] == nil) 
	then
		print("Invalid proposal")
		return false
	end

	-- do vote & restore
	proposal[proposalIndex]["voteCount"] =  proposal[proposalIndex]["voteCount"] + 1
	voters[sender]["voteTo"] = index

	L0.PutState("voters", voters)
	L0.PutState("proposal", proposal)
end

-- 获取票数最多的, 如有并列取第一个
function winner() 
	-- get proposal	
	local proposal = L0.GetState("proposal")
	--proposal = deserialize(proposal)

	local winnerIndex = 0	
	local mostVotes = 0
	for i, v in ipairs(proposal)
	do
		local voteCount = v["voteCount"]
		if voteCount > mostVotes
		then
			mostVotes = voteCount
			winnerIndex = i
		end
	end
	return winnerIndex	
end

-- 每次合约查询都调用
function L0Query(args)
	local key = args[0] 
	if key == "chairperson" then
		return L0.GetState("chairperson")
	end
	if key == "voters" then
		local voterAddr = args[1]
		local voters = L0.GetState("voters")
		local res = voterAddr..","..voters[voterAddr]["voteTo"]
		return res
	end
	if key == "winner" then
		return winner()
	end
	if key == "proposal" then
		return "proposal"
	end
end

