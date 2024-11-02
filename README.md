# Streakode: Developer Motivation & Insight Tool

**Streakode** is a shell plugin designed to motivate developers by tracking their Git activity and providing personalized insights, streak stats, and gentle reminders to keep coding fun and productive.
By analyzing commit data across your Git projects, Streakode helps you build consistency, stay engaged, and celebrate your achievements, all directly from your terminal.

## Features
- **Commit Streaks**: Track daily commit streaks and visualize your ongoing coding journey.
- **Activity Insights**: See your most active projects, most productive times, and commit patterns.
- **Achievements & Gamification**: Earn badges like “Early Bird” for morning coding, or “Night Owl” for late-night sessions.
- **Goal Tracking**: Set weekly or monthly commit goals and watch your progress.
- **Customizable Reminders**: Get motivational prompts if you miss a day or need a gentle nudge to keep coding.

## Installation
- To be determined.

## Usage
- **Show Stats**: `streakode stats` – Get an overview of commit streaks, top projects, and activity insights.
- **View Achievements**: `streakode achievements` – See your unlocked achievements and progress toward new ones.
- **Track Goals**: `streakode goals` – Set and view weekly or monthly commit goals.
- **Configuration**: Customize settings in `.streakodeconfig` to personalize your experience.
- **Reminders**: Get gentle reminders to code if you miss a day or fall behind on your goals.
- **Startup**: Automatically start Streakode with your shell to stay motivated every time you code.

## Contributing
Contributions are welcome! Fork the repository, make your changes, and submit a pull request. Suggestions for new features, gamification ideas, or performance improvements are always appreciated.

## License
MIT License.

---

# Development Plan for Streakode

## Project Overview
Streakode is a motivational and insights tool designed to encourage developers by tracking Git activity across projects. Written in Go, the tool integrates into the shell, using commit data to generate insights and gamified elements, helping users maintain streaks, track goals, and celebrate progress.

## Milestones & Key Features

### Milestone 1: Initial Setup & Git Scanning
- **Objective**: Set up the project and implement basic Git repository scanning.
- **Tasks**:
    - Initialize a Go project with a CLI structure.
    - Implement a function to scan specified directories for Git projects.
    - Write `streakode init` and `streakode scan` commands to manage project scanning.

### Milestone 2: Core Stats & Streak Tracking
- **Objective**: Develop functionality to track commit streaks, top projects, and other activity metrics.
- **Tasks**:
    - Implement commit streak calculation logic, storing streak length and start date.
    - Develop functions to rank projects by commit frequency and identify most active projects.
    - Calculate productivity insights, such as the most productive days and times.
    - Add a `streakode stats` command to display this data in the terminal.

### Milestone 3: Gamification & Motivation System
- **Objective**: Add motivational features, including achievements, goals, and reminders.
- **Tasks**:
    - Define and implement achievements (e.g., "Early Bird" or "Weekend Warrior").
    - Add goal-setting functionality, with user-defined weekly or monthly commit goals.
    - Write reminder logic to prompt users after periods of inactivity.
    - Create `streakode achievements` and `streakode goals` commands to interact with these features.

### Milestone 4: CLI Interface & Visual Feedback
- **Objective**: Improve the CLI experience with intuitive commands and visual enhancements.
- **Tasks**:
    - Add command options for stats, achievements, and goals, with clear help documentation.
    - Design ASCII-based visuals, such as progress bars for goals or badges for achievements.
    - Improve feedback and error handling, ensuring smooth user experience.

### Milestone 5: Customization & Configuration Management
- **Objective**: Allow users to customize paths, notifications, and goal preferences.
- **Tasks**:
    - Create a config file parser to handle directory paths, notification settings, and goals.
    - Integrate configuration settings into the main functionality, allowing personalized experience.
    - Ensure that config options can be modified easily in `.streakodeconfig`.

### Milestone 6: Testing & Documentation
- **Objective**: Ensure stability, usability, and provide thorough documentation.
- **Tasks**:
    - Write unit tests for Git scanning, streak calculation, and gamification logic.
    - Document setup, configuration options, and available CLI commands.
    - Create a user guide with examples, tips, and troubleshooting.

## Project Timeline
| Milestone                 | Duration | Target Completion |
|---------------------------|----------|-------------------|
| Initial Setup             | 1 week   | Month 1, Week 1   |
| Core Stats & Streaks      | 2 weeks  | Month 1, Week 3   |
| Gamification System       | 2 weeks  | Month 2, Week 1   |
| CLI Interface             | 1 week   | Month 2, Week 2   |
| Customization & Config    | 1 week   | Month 2, Week 3   |
| Testing & Documentation   | 1 week   | Month 2, Week 4   |

## Technical Stack
- **Language**: Go
- **Libraries**: Go-Git (for Git interaction), Cobra (for CLI structure)
- **Version Control**: GitHub for collaborative development

## Future Enhancements
1. **Weekly & Monthly ASCII Reports**: Add ASCII-based charts and graphs summarizing developer activity.
2. **Team Leaderboard**: Optionally compare achievements and streaks in a team setting.
3. **Notification Integration**: Enable optional notifications for mobile or desktop for better engagement.

## Potential Challenges
- **Git Parsing**: Efficiently handling large repositories and ensuring accurate commit data.
- **Motivation Balance**: Creating a gamification system that is encouraging without feeling overwhelming.
- **Config Customization**: Ensuring seamless configuration options that feel natural within the shell environment.

---
