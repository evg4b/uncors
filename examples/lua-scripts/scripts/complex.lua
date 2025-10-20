-- Complex Lua script demonstrating advanced features
local math = require("math")
local string = require("string")
local table = require("table")

-- Generate some complex data
local function generateData()
    math.randomseed(os.time())

    local data = {
        random_number = math.random(1, 1000),
        timestamp = os.date("%Y-%m-%d %H:%M:%S"),
        pi_approximation = math.floor(math.pi * 1000) / 1000,
    }

    return data
end

-- Format data as JSON
local function formatJSON(data)
    local parts = {}

    for key, value in pairs(data) do
        local valueStr
        if type(value) == "number" then
            valueStr = tostring(value)
        elseif type(value) == "string" then
            valueStr = '"' .. value .. '"'
        elseif type(value) == "boolean" then
            valueStr = tostring(value)
        else
            valueStr = 'null'
        end

        table.insert(parts, '"' .. key .. '": ' .. valueStr)
    end

    return '{' .. table.concat(parts, ', ') .. '}'
end

-- Main execution
local data = generateData()

-- Add request information
data.request_method = request.method
data.request_path = request.path
data.has_body = request.body ~= nil and request.body ~= ""

-- Build response
response.status = 200
response.body = formatJSON(data)
response.headers["Content-Type"] = "application/json"
response.headers["X-Script-Type"] = "file-based"
response.headers["X-Generated-At"] = os.date("%Y-%m-%d %H:%M:%S")
