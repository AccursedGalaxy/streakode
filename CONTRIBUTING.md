# Contributing to Streakode

First off, thank you for considering contributing to Streakode! It's people like you that make Streakode such a great tool.

## Code of Conduct

By participating in this project, you are expected to uphold our Code of Conduct:

- Use welcoming and inclusive language
- Be respectful of differing viewpoints and experiences
- Gracefully accept constructive criticism
- Focus on what is best for the community
- Show empathy towards other community members

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the issue list as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

* Use a clear and descriptive title
* Describe the exact steps which reproduce the problem
* Provide specific examples to demonstrate the steps
* Describe the behavior you observed after following the steps
* Explain which behavior you expected to see instead and why
* Include details about your configuration and environment

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

* Use a clear and descriptive title
* Provide a step-by-step description of the suggested enhancement
* Provide specific examples to demonstrate the steps
* Describe the current behavior and explain which behavior you expected to see instead
* Explain why this enhancement would be useful to most Streakode users

### Pull Requests

* Fill in the required template
* Do not include issue numbers in the PR title
* Follow the Go coding style
* Include appropriate test coverage
* End all files with a newline
* Write clear, descriptive commit messages

## Development Process

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code lints
6. Issue that pull request!

### Development Setup

```bash
# Clone your fork
git clone git@github.com:<your-username>/streakode.git

# Add upstream remote
git remote add upstream https://github.com/AccursedGalaxy/streakode.git

# Create your feature branch
git checkout -b feature/my-new-feature

# Install dependencies
make dev-deps
```

### Coding Style

* Use `gofmt` for formatting
* Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
* Write descriptive comments for non-obvious code sections
* Keep functions focused and modular
* Use meaningful variable names

### Testing

* Write unit tests for new features
* Ensure all tests pass before submitting PR
* Run tests with `make test`
* Aim for high test coverage on new code

### Documentation

* Update README.md with details of changes to the interface
* Update the wiki/docs for substantial changes
* Comment your code where necessary
* Keep documentation up to date with changes

## Git Commit Messages

* Use the present tense ("Add feature" not "Added feature")
* Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
* Limit the first line to 72 characters or less
* Reference issues and pull requests liberally after the first line
* Consider starting the commit message with an applicable emoji:
    * 🎨 `:art:` when improving the format/structure of the code
    * 🐎 `:racehorse:` when improving performance
    * 📝 `:memo:` when writing docs
    * 🐛 `:bug:` when fixing a bug
    * 🔥 `:fire:` when removing code or files
    * ✅ `:white_check_mark:` when adding tests
    * 🔒 `:lock:` when dealing with security
    * ⬆️ `:arrow_up:` when upgrading dependencies
    * ⬇️ `:arrow_down:` when downgrading dependencies

## Release Process

1. Update the version number in relevant files
2. Update the CHANGELOG.md
3. Create a new GitHub release with the version number
4. Write release notes detailing changes

## Questions?

Don't hesitate to reach out to the maintainers if you have questions. We're here to help! 