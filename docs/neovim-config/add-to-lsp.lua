local configs = require("lspconfig.configs")

configs.nobl9_language_server = {
  default_config = {
    cmd = { "nobl9-language-server" },
    filetypes = { "yaml" },
    root_dir = function(fname)
      return vim.fs.dirname(vim.fs.find(".git", { path = fname, upward = true })[1])
    end,
    settings = {},
  },
}
