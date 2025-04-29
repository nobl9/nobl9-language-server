local a = require("plenary.async.tests")
local uv = vim.uv

local server_cmd = "nobl9-language-server"
local test_files_dir = "tests/lua/inputs"

local M = {}

M.timeouts = {
  server_startup = 5000,
  request = 5000,
  diagnostics_publish = 10000,
  interval = 20,
}

--- Creates a new test buffer with the given lines and sets its filetype.
-- @param buf (string): The buffer contents as a single string.
-- @return bufnr (number): The buffer number of the created buffer.
function M.create_test_buffer(filename, buf)
  local lines = vim.split(buf, "\n")
  local bufnr = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
  vim.bo[bufnr].filetype = "yaml"
  if filename then
    vim.api.nvim_buf_set_name(bufnr, filename)
  end
  vim.api.nvim_win_set_buf(0, bufnr)
  return bufnr
end

--- Reads the contents of a test file from the tests/inputs/plenary directory and creates a test buffer.
-- @param filename (string): The name of the file to read.
-- @return bufnr (number): The buffer number of the created buffer.
function M.read_test_file(filename)
  local path = test_files_dir .. "/" .. filename
  local fd = assert(uv.fs_open(path, "r", 438))
  local stat = assert(uv.fs_fstat(fd))
  local content = assert(uv.fs_read(fd, stat.size, 0))
  uv.fs_close(fd)
  return M.create_test_buffer(path, content)
end

--- Starts the LSP client for testing.
-- @return client_id (number): The ID of the started LSP client.
function M.setup_lsp_client()
  local client_id = vim.lsp.start({
    name = "n9-lsp-client",
    cmd = { server_cmd, "--logFilePath=plenary-test.log", "--logLevel=TRACE" },
    root_dir = vim.fn.getcwd(),
  })
  assert(client_id, "Failed to start LSP client")
  return client_id
end

--- Waits for the LSP client to attach to the given buffer.
-- @param bufnr (number): The buffer number to check for LSP client attachment.
function M.wait_for_lsp(bufnr)
  vim.wait(M.timeouts.server_startup, function()
    local clients = vim.lsp.get_clients({ bufnr = bufnr })
    return #clients > 0
  end, M.timeouts.interval)
end

--- Runs an LSP request on a buffer at a given cursor position.
-- @param method (string): The LSP request method (e.g., "textDocument/hover").
-- @param cursor_pos (table): The cursor position as {line, col} (1-indexed).
-- @param bufnr (number): The buffer number to use for the request.
-- @return result (table): The result of the LSP request for the started client.
function M.run_request(method, cursor_pos, bufnr)
  -- Prepare.
  vim.api.nvim_win_set_cursor(0, cursor_pos)
  local client_id = M.setup_lsp_client()
  M.wait_for_lsp(bufnr)

  -- Run.
  local params = vim.lsp.util.make_position_params()
  local results = vim.lsp.buf_request_sync(bufnr, method, params, M.timeouts.request)

  -- Assert.
  assert(results and results[client_id], string.format("Expected result from method '%s', but got nil", method))
  return results[client_id].result
end

a.describe("Nobl9 LSP", function()
  a.it("hover", function()
    -- Prepare and run.
    local bufnr = M.read_test_file("hover.yaml")
    local result = M.run_request("textDocument/hover", { 1, 4 }, bufnr)

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
    local bufnr = M.read_test_file("completion.yaml")
    local result = M.run_request("textDocument/completion", { 1, 14 }, bufnr)

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
    local bufnr = M.read_test_file("diagnostics.yaml")
    M.setup_lsp_client()
    M.wait_for_lsp(bufnr)

    -- Wait for diagnostics to be published (may need to increase wait time)
    vim.wait(M.timeouts.diagnostics_publish, function()
      local diags = vim.diagnostic.get(bufnr)
      return #diags > 0
    end, M.timeouts.interval)

    local diagnostics = vim.diagnostic.get(bufnr)
    assert(diagnostics and #diagnostics > 0, "Expected diagnostics, but got none")
    local expected = {
      {
        bufnr = bufnr,
        col = 0,
        end_col = 8,
        end_lnum = 2,
        lnum = 2,
        message = "metadata.project: property is required but was empty",
        namespace = 3,
        severity = 1,
        source = "nobl9-language-server",
        user_data = {
          lsp = {
            message = "metadata.project: property is required but was empty",
            range = {
              ["end"] = {
                character = 8,
                line = 2,
              },
              ["start"] = {
                character = 0,
                line = 2,
              },
            },
            severity = 1,
            source = "nobl9-language-server",
          },
        },
      },
    }
    assert.are.same(expected, diagnostics)
  end)
end)
