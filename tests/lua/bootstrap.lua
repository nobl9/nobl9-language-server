local function println(...)
  io.write(table.concat({...}, "\t") .. "\n")
end

--- Initialize the test environment.
--- Thie file will run once before attempting to run PlenaryBustedDirectory.
vim.cmd([[set runtimepath=$VIMRUNTIME]]) -- reset, otherwise it contains all of $PATH
println("Runtime path: " .. vim.inspect(vim.opt.runtimepath:get()))
vim.opt.swapfile = false
local site_dir = "./tests/lua/.plenary-tests/all/site"
vim.opt.packpath = { site_dir }

local plugins = {
  ["plenary.nvim"] = {
    import = "plenary",
    url = "https://github.com/nvim-lua/plenary.nvim",
  },
}

for plugin, data in pairs(plugins) do
  local plugin_path = site_dir .. "/pack/deps/start/" .. plugin
  if vim.fn.isdirectory(plugin_path) ~= 1 then
    os.execute("git clone " .. data.url .. " " .. plugin_path)
  else
    println("Plugin " .. plugin .. " already downloaded")
  end
  println("Adding plugin to runtimepath: " .. plugin_path)
  vim.opt.runtimepath:append(plugin_path)

  require(data.import)
end

println("Runtime path: " .. vim.inspect(vim.opt.runtimepath:get()))
println("Package path: " .. package.path)

-- Check if PlenaryBustedDirectory command is available
vim.cmd([[runtime plugin/plenary.vim]])
if vim.fn.exists(":PlenaryBustedDirectory") == 0 then
  vim.notify("minimal_init.lua: Failed to find PlenaryBustedDirectory command. Aborting!", vim.log.levels.ERROR)
  vim.cmd("q!")
end
