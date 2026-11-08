local key = KEYS[1]
local window_ms = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])

local current_window = math.floor(now_ms / window_ms)
local current_window_start = current_window * window_ms
local elapsed_in_current = now_ms - current_window_start
local elapsed_fraction = elapsed_in_current / window_ms

local data = redis.call("HMGET", key, "window", "curr_count", "prev_count")
local saved_window = tonumber(data[1])
local curr_count = tonumber(data[2]) or 0
local prev_count = tonumber(data[3]) or 0

if not saved_window then
    saved_window = current_window
    curr_count = 0
    prev_count = 0
elseif saved_window ~= current_window then
    if current_window - saved_window == 1 then
        prev_count = curr_count
    else
        prev_count = 0
    end
    curr_count = 0
    saved_window = current_window
end

local estimated_count = (prev_count * (1.0 - elapsed_fraction)) + curr_count

if estimated_count >= limit then
    redis.call("PEXPIRE", key, window_ms * 2)
    return 0
end

curr_count = curr_count + 1
redis.call("HMSET", key, "window", saved_window, "curr_count", curr_count, "prev_count", prev_count)
redis.call("PEXPIRE", key, window_ms * 2)
return 1

