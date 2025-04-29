local a = require("plenary.async.tests")

local server_cmd = "nobl9-language-server"

local timeouts = {
  server_startup = 1000,
  request = 1000,
  diagnostics_publish = 1500,
  interval = 20,
}

--- Starts the LSP client for testing.
-- @return client_id (number): The ID of the started LSP client.
local function setup_lsp_client()
  local client_id = vim.lsp.start({
    name = "n9-lsp-client",
    cmd = { server_cmd, "--logFilePath=plenary-test.log", "--logLevel=TRACE" },
    root_dir = vim.fn.getcwd(),
  })
  assert(client_id, "Failed to start LSP client")
  return client_id
end

--- Creates a new test buffer with the given lines and sets its filetype.
-- @param buf (string): The buffer contents as a single string.
-- @param filetype (string): (Optional) Filetype to set for the buffer. Defaults to "yaml".
-- @return bufnr (number): The buffer number of the created buffer.
local function create_test_buffer(buf, filetype)
  local lines = vim.split(buf, "\n")
  local bufnr = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
  vim.bo[bufnr].filetype = filetype or "yaml"
  vim.api.nvim_win_set_buf(0, bufnr)
  return bufnr
end

--- Waits for the LSP client to attach to the given buffer.
-- @param bufnr (number): The buffer number to check for LSP client attachment.
local function wait_for_lsp(bufnr)
  vim.wait(timeouts.server_startup, function()
    local clients = vim.lsp.get_clients({ bufnr = bufnr })
    return #clients > 0
  end, timeouts.interval)
end

--- Runs an LSP request on a temporary buffer with the given contents.
-- @param method (string): The LSP request method (e.g., "textDocument/hover").
-- @param cursor_pos (table): The cursor position as {line, col} (1-indexed).
-- @param buf (string): The buffer contents as a single string.
-- @return result (table): The result of the LSP request for the started client.
local function run_request(method, cursor_pos, buf)
  -- Prepare.
  local bufnr = create_test_buffer(buf)
  vim.api.nvim_win_set_cursor(0, cursor_pos)
  local client_id = setup_lsp_client()
  wait_for_lsp(bufnr)

  -- Run.
  local params = vim.lsp.util.make_position_params()
  local results = vim.lsp.buf_request_sync(bufnr, method, params, timeouts.request)

  -- Assert.
  assert(
    results and results[client_id],
    string.format("Expected result from method '%s', but got nil", "textDocument/hover")
  )
  return results[client_id].result
end

a.describe("Nobl9 LSP", function()
  a.it("hover", function()
    -- Prepare and run.
    local buf = [[
  apiVersion: n9/v1alpha
  kind: Service
  metadata:
    name: my-service
  ]]
    local result = run_request("textDocument/hover", { 1, 4 }, buf)

    -- Assert.
    assert(result, "Expected hover result, but got nil")
    assert(result.contents, "Expected hover contents, but got none")
    local expected = {
      kind = "markdown",
      value = [[
`apiVersion:string`

Version represents the specific version of the manifest.

**Validation rules:**

- should be equal to 'n9/v1alpha']],
    }
    assert.are.same(expected, result.contents)
  end)

  a.it("completion", function()
    -- Prepare and run.
    local buf = [[
  apiVersion: n9/v1alpha
  kind: Service
  metadata:
    name: my-service
  ]]
    local result = run_request("textDocument/completion", { 1, 14 }, buf)

    -- Assert.
    assert(result, "Expected completion result, but got nil")
    local expected = {
      {
        kind = 12,
        label = "n9/v1alpha",
      },
    }
    assert.are.same(expected, result)
  end)

  a.it("diagnostics", function()
    -- Prepare and run.
    local buf = [[
  apiVersion: n9/v1alpha
  kind: Service
  ]]
    local bufnr = create_test_buffer(buf)
    setup_lsp_client()
    wait_for_lsp(bufnr)

    -- Wait for diagnostics to be published (may need to increase wait time)
    vim.wait(timeouts.diagnostics_publish, function()
      local diags = vim.diagnostic.get(bufnr)
      return #diags > 0
    end, timeouts.interval)

    local diagnostics = vim.diagnostic.get(bufnr)
    assert(diagnostics and #diagnostics > 0, "Expected diagnostics, but got none")
  end)
end)
