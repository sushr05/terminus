local key = KEYS[1]
local window_ms = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local member = ARGV[4]

local cutoff = now_ms - window_ms

redis.call("ZREMRANGEBYSCORE", key, "-inf", cutoff)

local current_count = redis.call("ZCARD", key)

if current_count >= limit then
    redis.call("PEXPIRE", key, window_ms)
    return 0
end

redis.call("ZADD", key, now_ms, member)
redis.call("PEXPIRE", key, window_ms)
return 1

