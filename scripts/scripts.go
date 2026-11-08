package scripts

import _ "embed"

//go:embed redis_fixed_window.lua
var RedisFixedWindow string

//go:embed redis_token_bucket.lua
var RedisTokenBucket string

//go:embed redis_sliding_log.lua
var RedisSlidingLog string

//go:embed redis_sliding_counter.lua
var RedisSlidingCounter string

//go:embed redis_leaky_bucket.lua
var RedisLeakyBucket string
