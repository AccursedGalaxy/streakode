# Streakode 🚀

[![Release](https://img.shields.io/github/v/release/AccursedGalaxy/streakode)](https://github.com/AccursedGalaxy/streakode/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/AccursedGalaxy/streakode)](https://goreportcard.com/report/github.com/AccursedGalaxy/streakode)
[![License](https://img.shields.io/github/license/AccursedGalaxy/streakode?color=blue)](https://github.com/AccursedGalaxy/streakode/blob/main/LICENSE.md)

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

Proper Installation Flow Coming Soon.
-> Including automatic config file creation and more.

### From Releases

Download the latest release for your platform from the [releases page](https://github.com/AccursedGalaxy/streakode/releases).

### Building from Source

```bash
git clone https://github.com/AccursedGalaxy/streakode.git
cd streakode
make install
```

## Configuration 📝

First, check your Git author configuration:

```bash
streakode author
```

This will show your global and local Git configurations. Use the displayed name in your Streakode config file.

Create a configuration file at `~/.streakodeconfig.yaml`:

```yaml
# Use the name exactly as shown by 'streakode author' command
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

colors:
  header_color: "#FF96B4"
  section_color: "#87CEEB"
  divider_color: "#808080"
```

> **Important**: The `author` field must match your Git author name exactly as it appears in your Git configuration. This ensures Streakode only tracks commits made by you. Use `streakode author` to verify your Git author name.

### Multiple Profiles

You can create different profiles by creating additional config files:
- Work profile: `~/.streakodeconfig_work.yaml`
- Home profile: `~/.streakodeconfig_home.yaml`

Each profile can have a different author configuration if needed (useful if you use different Git identities for work/personal projects).

## Usage 💻

### Basic Commands

```bash
# Show version
streakode --version
streakode version

# Check your Git author configuration
streakode author

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

Or if you cloned the repository, you can pull the latest changes and build the project again.

```bash
git pull
make clean
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

### Next To Implement 🚀

#### 1. Enhanced Historical Data Collection 📊

Expanding data collection capabilities to provide deeper insights into coding patterns and productivity trends.

**Key Improvements:**
- Track commit metadata historically for advanced analytics
  - Upgrade Current `RepoMetadata` struct to include more data.
  - Implement Historical data stroage using the current caching system (may need upgrades) 
- Analyze peak coding hours and most productive days
  - Add new functions in the scanning logic to calculte:
    - peak coding hours - save historical snapshots
    - most producive days
    - coding velocity
- Monitor velocity and metrics changes over time
  - Allows for a more in depth and egaging stats display

**Implementation Details:**
1. Extend `RepoMetadata` structure:
   - Add arrays for daily/hourly commit tracking
   - Enable timestamp-based commit analysis
   - Store historical snapshots to minimize repository rescanning

2. Velocity Trend Analysis:
   - Calculate daily commit velocity (7/30 day averages)
   - Compare current vs historical velocities
   - Store metrics in new `DailyCommitData` structure

#### 2. Goal Tracking & Progress Visualization 🎯

Enhance the user experience with visual progress tracking and goal management.

**Planned Features:**
- ASCII progress bars for weekly/monthly goals
- Percentage-based completion indicators
- Integration with user-defined goals from config
- Real-time progress tracking against set targets

#### 3. Enhanced Profile Management 👤

Streamline profile management with new CLI command

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