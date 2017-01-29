How to Contribute
=================

Inbucket encourages third-party patches. It's valuable to know how other
developers are using the product.


## Getting Started

If you anticipate your issue requiring a large patch, please first submit a
GitHub issue describing the problem or feature. You are also encouraged to
outline the process you would like to use to resolve the issue. I will attempt
to provide validation and/or guidance on your suggested approach.


## Making Changes

Inbucket uses [git-flow] with default options.  If you have git-flow installed,
you can run `git flow feature start <topic branch name>`.

Without git-flow, create a topic branch from where you want to base your work:
  - This is usually the `develop` branch, example command:
    `git checkout origin/develop -b <topic branch name>`
  - Only target the `master` branch if the issue is already resolved in
    `develop`.

Once you are on your topic branch:

1. Make commits of logical units.
2. Add unit tests to exercise your changes.
3. Run the updated code through `go fmt` and `go vet`.
4. Ensure the code builds and tests with the following commands:
  - `go clean ./...`
  - `go build ./...`
  - `go test ./...`


## Thanks

Thank you for contributing to Inbucket!

[git-flow]: https://github.com/nvie/gitflow
