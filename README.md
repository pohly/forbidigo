# forbidigo

`forbidigo` is a Go static analysis tool to forbidigo use of particular identifiers.

`forbidigo` is recommended to be run as part of [golangci-lint](https://github.com/golangci/golangci-lint) where it can be controlled using file-based configuration and `//nolint` directives, but it can also be run as a standalone tool.

## Installation

    go get -u github.com/ashanbrown/forbidigo

## Usage

    forbidigo [flags...] patterns... -- packages...

If no patterns are specified, the default pattern of `^(fmt\.Print.*|print|println)$` is used to eliminate debug statements.  By default,
functions (and whole files), that are identifies as Godoc examples (https://blog.golang.org/examples) are excluded from 
checking.

A larger set of interesting patterns might include:

* `^fmt\.Print.*$` -- forbid use of Print statements because they are likely just for debugging
* `^fmt\.Errorf$` -- forbid Errorf in favor of using github.com/pkg/errors
* `^ginkgo\.F[A-Z].*$` -- forbid ginkgo focused commands (used for debug issues)
* `^spew\.Dump$` -- forbid dumping detailed data to stdout
* `^fmt\.Errorf(# please use github\.com/pkg/errors)?$` -- forbid Errorf, with a custom message

Note that the linter has no knowledge of what packages were actually imported, so aliased imports will match these patterns.

### Flags
- **-set_exit_status** (default false) - Set exit status to 1 if any issues are found.
- **-exclude_godoc_examples** (default true) - Controls whether godoc examples are identified and excluded
- **-tests** (default true) - Controls whether tests are included

## Purpose

To prevent leaving format statements and temporary statements such as Ginkgo FIt, FDescribe, etc.

## Ignoring issues

You can ignore a particular issue by including the directive `//permit` on that line.  *This feature is disabled inside `golangci-lint` to encourage ignoring issues using the `// nolint` directive common for all linters (nolinting well is hard and I didn't want to make an effort do it exactly right within this linter).*

## Contributing

Pull requests welcome!
