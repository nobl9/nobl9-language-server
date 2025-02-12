vim.g.mapleader = " "

vim.api.nvim_exec(
  [[
let g:UltiSnipsExpandTrigger='<Tab>'
let g:UltiSnipsJumpForwardTrigger='<c-j>'
let g:UltiSnipsJumpBackwardTrigger='<c-k>'
]],
  false
)

local lazypath = vim.loop.cwd() .. "neovim-config/.lazy/lazy.nvim"
if not vim.loop.fs_stat(lazypath) then
  vim.fn.system({
    "git",
    "clone",
    "--filter=blob:none",
    "https://github.com/folke/lazy.nvim.git",
    "--branch=stable", -- latest stable release
    lazypath,
  })
end
vim.opt.rtp:prepend(lazypath)

require("lazy").setup({
  "neovim/nvim-lspconfig",
  {
    "hrsh7th/nvim-cmp",
    dependencies = {
      "hrsh7th/cmp-nvim-lsp",
      { "Sirver/ultisnips", event = { "InsertEnter" } },
    },
    config = function()
      local cmp = require("cmp")
      cmp.setup({
        sources = {
          { name = "nvim_lsp" },
          { name = "ultisnips" },
        },
        mapping = cmp.mapping.preset.insert({
          ["<C-b>"] = cmp.mapping.scroll_docs(-4),
          ["<C-f>"] = cmp.mapping.scroll_docs(4),
          ["<C-Space>"] = cmp.mapping.complete(),
          ["<C-e>"] = cmp.mapping.abort(),
          ["<CR>"] = cmp.mapping.confirm({ select = true }),
        }),
        snippet = {
          expand = function(args)
            vim.fn["UltiSnips#Anon"](args.body) -- For `ultisnips` users.
            -- vim.snippet.expand(args.body) -- For native neovim snippets (Neovim v0.10+)
          end,
        },
      })
    end,
  },
})

local lsp = require("lspconfig")
local configs = require("lspconfig.configs")

local function keymap(bufnr, _)
  -- See `:help vim.lsp.*` for documentation on any of the below functions
  local nmap = function(keys, func, desc)
    if desc then
      desc = "LSP: " .. desc
    end
    vim.keymap.set("n", keys, func, { buffer = bufnr, desc = desc })
  end

  nmap("<leader>ca", vim.lsp.buf.code_action, "[C]ode [A]ction")
  nmap("<leader>fm", vim.lsp.buf.format, "[F]or[M]at buffer")
  nmap("K", vim.lsp.buf.hover, "Hover Documentation")
end

configs.nobl9-language-server = {
  default_config = {
    cmd = { "nobl9-language-server", "-logLevel=TRACE" },
    filetypes = { "yaml" },
    root_dir = function(fname)
      return lsp.util.find_git_ancestor(fname)
    end,
    settings = {},
  },
}

local capabilities = require("cmp_nvim_lsp").default_capabilities()
lsp.nobl9-language-server.setup({
  on_attach = function(_, bufnr)
    keymap(bufnr)
  end,
  capabilities = capabilities,
  message_level = vim.lsp.protocol.MessageType.Info,
})
