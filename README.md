# awsutils
Commonly used aws utilities...


<img src="unsplash.jpg" alt="Photo by Todd Quackenbush at Unsplash" />

## Project Health Status

![CI](https://github.com/prognoshealth/awsutils/workflows/CI/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/prognoshealth/awsutils)](https://goreportcard.com/report/github.com/prognoshealth/awsutils) [![Maintainability](https://api.codeclimate.com/v1/badges/caf656491b33b31a018e/maintainability)](https://codeclimate.com/github/prognoshealth/awsutils/maintainability) [![Test Coverage](https://api.codeclimate.com/v1/badges/caf656491b33b31a018e/test_coverage)](https://codeclimate.com/github/prognoshealth/awsutils/test_coverage)



## Build, Test and Lint

```sh
> go build ./...
> go test -cover ./...
> golangci-lint run
```

## Contributing

Thanks for contributing!

### Git Workflow

Use a squashed feature branch development workflow, for example:

```sh
> git pull origin master
> git checkout -b feature-branch-name

# ... do work and make commits...
# with pushes to remote and github pull requests...

> git fetch
> git rebase -i origin/master
> git push --force origin feature-branch-name
> git checkout master
> git pull origin master
> git merge feature-branch-name
> git push origin master
```

For commit comments use the form:

> Description of feature or change being made.

For more details on what makes a good commit message check out
https://chris.beams.io/posts/git-commit/.

#### Some Reading Material

- https://blog.carbonfive.com/2017/08/28/always-squash-and-rebase-your-git-commits/
- https://www.atlassian.com/git/articles/git-team-workflows-merge-or-rebase