local lsp = require("lspconfig")
local configs = require("lspconfig.configs")

configs.nobl9-language-server = {
  default_config = {
    cmd = { "nobl9-language-server", "-logLevel=debug" },
    filetypes = { "yaml" },
    root_dir = function(fname)
      return lsp.util.find_git_ancestor(fname)
    end,
    settings = {},
  },
}
