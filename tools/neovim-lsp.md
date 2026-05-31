# Neovim Tooling

Install these LSPs/formatters for this repository:

```bash
go install golang.org/x/tools/gopls@latest
go install github.com/google/yamlfmt/cmd/yamlfmt@latest
go install github.com/rhysd/actionlint/cmd/actionlint@latest
npm install -g prettier prettier-plugin-sh yaml-language-server
```

Recommended Mason packages:

```text
gopls
yaml-language-server
yamlfmt
prettier
shellcheck
shfmt
actionlint
markdownlint
golangci-lint
```

`just fmt` is the repo-level formatter entrypoint. Go code is formatted by `go fmt`, Justfiles by
`just --fmt`, YAML by `yamlfmt`, and Markdown/JSON/JSONC by Prettier.
