--- Initialize before running each test.
vim.cmd([[set runtimepath=$VIMRUNTIME]]) -- reset, otherwise it contains all of $PATH
vim.opt.swapfile = false
vim.opt.packpath = { "./tests/.plenary-tests/all/site" } -- set packpath to the site directory
