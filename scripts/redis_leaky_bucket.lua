local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local leak_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local ttl = math.ceil(tonumber(ARGV[4]))

local data = redis.call("HMGET", key, "level", "last_check")
local level = tonumber(data[1])
local last_check = tonumber(data[2])

if not level or not last_check then
    level = 0
    last_check = now
else
    local elapsed = now - last_check
    if elapsed > 0 then
        level = math.max(0, level - (elapsed * leak_rate))
        last_check = now
    end
end

if level + 1 > capacity then
    redis.call("HMSET", key, "level", level, "last_check", last_check)
    redis.call("EXPIRE", key, ttl)
    return 0
end

level = level + 1
redis.call("HMSET", key, "level", level, "last_check", last_check)
redis.call("EXPIRE", key, ttl)
return 1

