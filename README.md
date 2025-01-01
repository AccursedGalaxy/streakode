<div align="center">

# Streakode 🚀

[![Release](https://img.shields.io/github/v/release/AccursedGalaxy/streakode)](https://github.com/AccursedGalaxy/streakode/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/AccursedGalaxy/streakode)](https://goreportcard.com/report/github.com/AccursedGalaxy/streakode)
[![License](https://img.shields.io/github/license/AccursedGalaxy/streakode?color=blue)](https://github.com/AccursedGalaxy/streakode/blob/main/LICENSE.md)

</div>

Streakode is a powerful Git analytics and search tool that combines advanced commit history exploration with insightful coding statistics. Perfect for developers who want to understand their coding patterns, search through their Git history efficiently, and maintain productive coding streaks across multiple repositories.

<div align="center">
  <img src="images/preview.png" alt="Streakode Preview" width="600">
</div>

## Key Features ✨

### 🔍 Advanced Git History Search
- Lightning-fast commit history search powered by `fzf` and `ripgrep`
- Interactive fuzzy search with real-time preview
- Rich commit details including diffs, stats, and branch information
- Smart repository detection and automatic GitHub links
- Efficient caching system for quick subsequent searches

### 📊 Comprehensive Analytics
- Detailed commit patterns and coding streaks
- Code changes tracking (+/-) per repository
- Language statistics and peak coding hours
- Project engagement metrics
- Weekly and monthly activity summaries

### 🎯 Developer Productivity Tools
- Weekly commit goals and progress tracking
- Multiple profile support (work/personal separation)
- Customizable activity indicators
- Smart caching for fast repository scanning
- Selective cache updates and version-aware management

### 🎨 Customizable Experience
- Modern, colorful terminal UI
- Configurable table layouts and themes
- Multiple insight views
- Flexible display options
- Profile-specific configurations

## Features ✨

- 📊 Enhanced commit tracking and statistics
  - Detailed weekly/monthly commit patterns
  - Code changes tracking (+/-) per repository
  - Language statistics and peak coding hours
  - Customizable activity indicators
- 🎨 Highly configurable display options
  - Customizable table layouts and styles
  - Color themes support
  - Multiple insight views
- 🔄 Smart caching system
  - Efficient repository scanning
  - Selective cache updates
  - Version-aware cache management
- 🎯 Advanced goal tracking
  - Weekly commit goals
  - Progress visualization
  - Customizable thresholds
- 👤 Profile management
  - Work/personal separation
  - Profile-specific configurations
  - Easy profile switching

## Installation 🛠️

### Prerequisites
- Go 1.19 or higher
- Git
- [fzf](https://github.com/junegunn/fzf) (required for interactive search)
- [ripgrep](https://github.com/BurntSushi/ripgrep) (optional, enhances search capabilities)

### From Releases

Download the latest release for your platform from the [releases page](https://github.com/AccursedGalaxy/streakode/releases).

### Building from Source

```bash
git clone https://github.com/AccursedGalaxy/streakode.git
cd streakode
make install
```

## Usage 💻

### Basic Commands

```bash
# Show version information
streakode version

# View Git author configuration
streakode author

# Display repository statistics
streakode stats [repository]

# Interactive commit history search
streakode history search

# Advanced commit search with filters
streakode history search --author="name" --since="2 weeks ago"

# Repository cache management
streakode cache reload  # Refresh cache
streakode cache clean   # Clear cache

# Profile management
streakode profile work    # Switch to work profile
streakode profile home    # Switch to home profile
```

### Interactive Search Features

The `history search` command provides powerful interactive search capabilities:
- Fuzzy search through commit history
- Real-time commit preview with diff
- Branch information and GitHub links
- File change statistics
- Multiple selection support
- Advanced filtering options

### Debug Mode

Enable debug mode with `--debug` or `-d` flag for any command:
```bash
streakode history search --debug
streakode stats --debug
```

This shows additional information about:
- Search parameters and filters
- Cache operations
- Configuration details
- Performance metrics

## Configuration 📝

Create a configuration file at `~/.streakodeconfig.yaml`. See the [example configuration](.defaultconfig.yaml) for all available options.

Key configuration sections:
```yaml
# Author and scanning settings
author: "YourName"
scan_directories:
  - "~/github"
  - "~/work/repos"

# Search and display settings
search_settings:
  max_results: 1000
  cache_timeout: 3600
  use_fuzzy_search: true

# UI customization
display_settings:
  theme: "modern"
  color_scheme: "dark"
  show_previews: true
```

## Updating 🔄

Logic for updating is currently not implemented.

If you want to update, you can manually download the latest release from the [releases page](https://github.com/AccursedGalaxy/streakode/releases) and replace the current binary.

Or if you cloned the repository, you can pull the latest changes and build the project again.

```bash
git pull
make clean
make build
make install
```

## Uninstallation 🗑️

To completely remove Streakode from your system:

```bash
# Remove the binary
go clean -i github.com/AccursedGalaxy/streakode

# Remove configuration and cache files
rm ~/.streakodeconfig*.yaml    # Removes all config files including profiles
rm ~/.streakode*.cache         # Removes all cache files including profiles
rm ~/.streakode.state          # Removes the state file

# Single Command For All Files and Configs (Linux/MacOS)
rm /usr/local/bin/streakode && rm ~/.streakodeconfig* ~/.streakode*.cache ~/.streakode.state

# For Windows users (PowerShell):
Remove-Item "$env:USERPROFILE\.streakodeconfig*.yaml"
Remove-Item "$env:USERPROFILE\.streakode*.cache"
Remove-Item "$env:USERPROFILE\.streakode.state"
```

Note: If you installed Streakode from a release binary instead of `go install`, simply delete the binary and the configuration files as shown above.

## Development 🛠️

### Setting Up Development Environment

```bash
# Clone the repository
git clone https://github.com/AccursedGalaxy/streakode.git
cd streakode

# Install development dependencies
make dev-deps

# Run tests
make test

# Build for development
make dev
```

### Project Structure

```
streakode/
├── cmd/          # Command implementations
├── cache/        # Caching system
├── config/       # Configuration management
├── scan/         # Repository scanning
└── search/       # Search functionality
```

### Contributing 🤝

Contributions are welcome! Here's how you can help:

1. Fork the repository
2. Create your feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

Please read our [Contributing Guidelines](CONTRIBUTING.md) for details on our code of conduct and development process.

## Roadmap 🗺️

- [ ] Enhanced search capabilities with advanced filters
- [ ] Team analytics and collaboration features
- [ ] Integration with CI/CD platforms
- [ ] Machine learning-based commit pattern analysis
- [ ] Custom plugin system
- [ ] Web interface (planned)

## Support 💖

If you find Streakode helpful:
- Give it a ⭐ on GitHub
- Share it with your network
- [Report issues](https://github.com/AccursedGalaxy/streakode/issues) or contribute
- Follow the project for updates

## License 📄

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments 🙏

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [fzf](https://github.com/junegunn/fzf) - Fuzzy finder
- [ripgrep](https://github.com/BurntSushi/ripgrep) - Fast search

Special thanks to all contributors who have helped shape Streakode into what it is today.

---

<div align="center">
  Made with ❤️ by <a href="https://github.com/AccursedGalaxy">AccursedGalaxy</a>
  <br>
  <sub>A powerful tool for developers who care about their Git history</sub>
</div>
