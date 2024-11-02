# Streakode 🚀

[![Release](https://img.shields.io/github/v/release/AccursedGalaxy/streakode)](https://github.com/AccursedGalaxy/streakode/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/AccursedGalaxy/streakode)](https://goreportcard.com/report/github.com/AccursedGalaxy/streakode)
[![License](https://img.shields.io/github/license/AccursedGalaxy/streakode)](https://github.com/AccursedGalaxy/streakode/blob/main/LICENSE)

Streakode is a powerful Git activity tracker that helps developers monitor their coding streaks, commit patterns, and project engagement. It provides insightful statistics about your coding habits across multiple repositories, with support for different profiles (e.g., work and personal projects).

## Features ✨

- 📊 Track commit streaks and coding patterns
- 🔄 Multiple profile support (work, personal, etc.)
- 📈 Weekly, monthly, and total commit statistics
- ⚡ Fast, cached repository scanning
- 🎯 Goal tracking and insights
- 🏠 Expandable home directory paths (~/)
- 💾 Persistent profile state management

## Installation 🛠️

### Using Go

```bash
go install github.com/AccursedGalaxy/streakode@latest
```

### From Releases

Download the latest release for your platform from the [releases page](https://github.com/AccursedGalaxy/streakode/releases).

### Building from Source

```bash
git clone https://github.com/AccursedGalaxy/streakode.git
cd streakode
make install
```

## Configuration 📝

Create a configuration file at `~/.streakodeconfig.yaml`:

```yaml
author: "Your Name"
dormant_threshold: 90  # days
scan_directories:
  - "~/github"
  - "~/work/projects"
refresh_interval: 24   # hours

display_stats:
  show_weekly_commits: true
  show_monthly_commits: true
  show_total_commits: true
  show_active_projects: true
  show_insights: true
  max_projects: 5

goal_settings:
  weekly_commit_goal: 10
```

### Multiple Profiles

You can create different profiles by creating additional config files:
- Work profile: `~/.streakodeconfig_work.yaml`
- Home profile: `~/.streakodeconfig_home.yaml`

## Usage 💻

### Basic Commands

```bash
# Show version
streakode --version
streakode version

# Check and install updates
streakode update

# Show statistics
streakode stats

# Refresh repository cache
streakode refresh

# Switch profiles
streakode profile work    # Switch to work profile
streakode profile home    # Switch to home profile
streakode profile -       # Switch to default profile

# Use different profile for a single command
streakode stats --profile work
```

## Updating 🔄

Logic for updating is currently not implemented.

If you want to update, you can manually download the latest release from the [releases page](https://github.com/AccursedGalaxy/streakode/releases) and replace the current binary.

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

### Example Output

```
🚀 Your Name's Coding Activity
──────────────────────────────────────
📊 3 commits this week • 12 this month • 156 total
──────────────────────────────────────
⚡ awesome-project: 2↑ this week • 🔥 2 day streak • today
⚡ cool-app: 1↑ this week • 2 days ago
──────────────────────────────────────
💫 awesome-project is your most active project with a 2 day streak!
```

## Contributing 🤝

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/AccursedGalaxy/streakode.git

# Install dependencies
go mod download

# Build and install locally
make dev
```

## License 📄

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments 🙏

- Thanks to all contributors who have helped shape Streakode
- Built with [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper)

## Support 💖

If you find Streakode helpful, please consider:
- Giving it a ⭐ on GitHub
- Sharing it with others
- [Reporting issues](https://github.com/AccursedGalaxy/streakode/issues) if you find any bugs

---

Made with ❤️ by [AccursedGalaxy](https://github.com/AccursedGalaxy)