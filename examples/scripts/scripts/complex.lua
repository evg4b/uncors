local function executeAndGetOutput(command)
    local fileHandle = assert(io.popen(command, 'r'))
    local output = fileHandle:read('*a')
    fileHandle:close() -- It's important to close the file handle
    return output
end

-- Build response
response.status = 200
response.body = executeAndGetOutput("fakedata --format=ndjson --limit 1 login=email referral=domain country=country")
response.headers["Content-Type"] = "application/json"
response.headers["X-Script-Type"] = "file-based"
response.headers["X-Generated-At"] = os.date("%Y-%m-%d %H:%M:%S")
