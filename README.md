# Streakode 🚀

[![Release](https://img.shields.io/github/v/release/AccursedGalaxy/streakode)](https://github.com/AccursedGalaxy/streakode/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/AccursedGalaxy/streakode)](https://goreportcard.com/report/github.com/AccursedGalaxy/streakode)
[![License](https://img.shields.io/github/license/AccursedGalaxy/streakode?color=blue)](https://github.com/AccursedGalaxy/streakode/blob/main/LICENSE.md)

Streakode is a powerful Git activity tracker that helps developers monitor their coding streaks, commit patterns, and project engagement. It provides insightful statistics about your coding habits across multiple repositories, with support for different profiles (e.g., work and personal projects).

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

Create a configuration file at `~/.streakodeconfig.yaml`:

```yaml
author: "Your Name"
dormant_threshold: 90
scan_directories:
  - "~/github"
  - "~/work/projects"
refresh_interval: 24

display_stats:
  show_welcome_message: true
  show_weekly_commits: true
  show_monthly_commits: true
  show_total_commits: true
  show_active_projects: true
  show_insights: true
  max_projects: 5
  
  # Table styling
  table_style:
    show_border: false
    column_separator: " "
    center_separator: "─"
    header_alignment: "center"
    show_header_line: false
    show_row_lines: false
    min_column_widths:
      repository: 20
      weekly: 8
      streak: 8
      changes: 13
      activity: 10

  # Activity indicators
  activity_indicators:
    high_activity: "🔥"
    normal_activity: "⚡"
    no_activity: "💤"
    streak_record: "🏆"
    active_streak: "🔥"

  # Activity thresholds
  thresholds:
    high_activity: 10

  # Insight settings
  insight_settings:
    top_languages_count: 3
    show_daily_average: true
    show_top_languages: true
    show_peak_coding: true
    show_weekly_summary: true
    show_weekly_goal: true
    show_most_active: true

goal_settings:
  weekly_commit_goal: 10

colors:
  header_color: "#FF96B4"
  section_color: "#87CEEB"
  divider_color: "#808080"

detailed_stats: true

language_settings:
  excluded_extensions: [".yaml", ".txt", ".md"]
  excluded_languages: ["YAML", "Text", "Markdown"]
  minimum_lines: 100
```

### Example Output

```
🚀 Your Name's Coding Activity
──────────────────────────────────────
📊 3 commits this week • 12 this month
──────────────────────────────────────
Repository    Weekly    Streak    Changes      Activity
🔥 project-a    5↑       3d🔥      +150/-50    today
⚡ project-b    2↑       1d        +80/-20     2d ago
💤 project-c    0↑       0d        +0/-0       5d ago
──────────────────────────────────────
📈 Weekly Summary: 7 commits, +230/-70 lines
📊 Daily Average: 1.0 commits
💻 Top Languages: Go:2.5k, Python:1.2k, JavaScript:0.8k
⏰ Peak Coding: 14:00-15:00 (3 commits)
🎯 Weekly Goal: 70% (7/10 commits)
```

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

#### 1. **Enhanced Historical Data Collection 📊**

   Deepen insights by tracking coding patterns, peak productivity times, and velocity trends.

   **Key Improvements:**
   - Track peak coding hours, productive days, and commit velocity over time.
   - Store historical snapshots to reduce rescanning.
   - Suggest productivity optimizations based on low-activity and high-efficiency periods.
   - Add achievements and basic productivity suggestions.

---

#### 2. **Goal Tracking & Visualization 🎯**

   Engage users with visual goal tracking and real-time progress updates.

   **Features:**
   - ASCII progress bars for weekly/monthly goals.
   - Percentage completion indicators for quick progress checks.
   - Track goal completion history for motivation.

   **Implementation:**
   - Integrate visual indicators into `DisplayStats`.
   - Track and display goal completion percentages in real-time.

---

#### 3. **Enhanced Profile Management 👤**

   Simplify profile switching and add preset configurations for quick setups.

   **Improvements:**
   - Preset profiles (e.g., `minimal`, `detailed`, `team-focused`).
   - Extended `profile` command for listing and setting defaults.
   - Dynamic config reload to apply changes without restarting.

---

#### 4. **Data Export and Team Reports 📈**

   Allow data export and team comparison features.

   **Commands:**
   - `streakode report`: Export detailed reports (e.g., weekly/monthly).
   - `streakode team velocity`: Compare coding velocity across team members.

   **Implementation:**
   - Add export options (Markdown, CSV).
   - Enable optional team velocity comparison for shared repositories.

---

#### 5. **Team Collaboration Features 👥**

   Make Streakode team-friendly by adding collaboration tools for developer groups.

   **Features:**
   - **Shared Reports**: Generate combined team reports for shared repositories.
   - **Team Velocity Tracking**: Track each team member’s velocity and contributions to the same project.
   - **Leaderboard**: Display a leaderboard view of team member contributions for motivation.
   - **Group Goals**: Set shared commit goals for team projects.

   **Implementation:**
   - Introduce a `team` configuration option to specify team members and shared projects.
   - Enhance `team velocity` to pull in data across configured team repositories.
   - Use a shared file (e.g., JSON/CSV) to aggregate data from team members.
   - Add privacy options to let users control visibility of their personal stats in team reports.

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