# Streakode Feature Implementation Plan 🚀

This document outlines new and additional features that are planned or already in development.

---

## IDEAS
- add optional "author" argument to the author command
   -> i.e.: streakode author AccursedGalaxy -> to get detailed stats about this author (currently just shows author set from the config to verify it's setup correctly, but can easily expand this to show detailed stats abou the author as well)

- new command set "history"
   -> i.e.: streakode history -a AccursedGalaxy -> get detailed and nicely formatted information about the recent git activity from the selected author (-a flag) going beyond just commit message. (I love the idea with a nice and insightful overview this gonna be usefull)
   -> i.e.: streakode hisotry -r streakode -> get detailed and nicely formatted information about the recent activity from a project not just form a single author. (Also really great idea and possibly super usefull)


1. **Branch-Based Search**:
```bash
streakode history branch <branch-name>
```
- Search commits specific to a branch across repos
- Show branch relationships and merge history
- Compare branches visually

2. **Content Search**:
```bash
streakode history search "query"
```
- Full-text search in commit diffs
- Search for specific code changes
- Find when a particular line was added/removed

3. **Patch Management**:
```bash
streakode history patch
```
- Interactive selection of commits to create patches
- Cherry-pick commits across repositories
- Create patch series for code review

4. **Time-Based Navigation**:
```bash
streakode history timeline
```
- Visual timeline of commits
- Group by time periods (day/week/month)
- Show parallel development streams

5. **Related Commits**:
```bash
streakode history related <commit-hash>
```
- Find commits that modify the same files
- Show commit dependencies
- Track bug fixes and related changes

6. **Tag/Release History**:
```bash
streakode history releases
```
- Show version tags and releases
- Group commits by release
- Show changelog-style history

7. **Refactor History**:
```bash
streakode history refactor <file/directory>
```
- Track major refactoring changes
- Show file movement history
- Visualize code evolution


## Next To Implement 🚀

### Testing Implementation
**Simulate Commit Data for Tests:**
Develop a method to generate synthetic Git commit data for various testing scenarios.
This will involve scripting the creation of mock repositories with diverse commit histories, authors, dates, and file changes.

By simulating different patterns and edge cases, I can thoroughly test the CLI's functionality, performance, and robustness without relying on real repositories.

**Detailed Implementation Plan:**
See [sim_testing.md](docs/sim_testing.md) for the complete testing implementation plan and details.


### 1. **Enhanced Historical Data Collection 📊**

   Gain deeper insights by capturing more historical data and identifying coding trends and productivity patterns.

   **Key Improvements:**
   - Track commit velocity over time to observe productivity shifts.
   - Provide productivity suggestions based on user-specific data trends, like identifying low-activity periods or highlighting peak productivity times.
   - Introduce achievements for milestones and offer basic productivity tips for improvement.

---

### 2. **Goal Tracking & Visualization 🎯**

   Boost engagement with visual goal tracking, allowing users to set, track, and achieve coding milestones.

   **Features:**
   - Progress bars and indicators for weekly/monthly goals with visual feedback.
   - Goal tracking with percentage completion to easily monitor progress.
   - Goal completion history for motivation, tracking personal bests and progress over time.

   **Implementation:**
   - Integrate real-time visual indicators into `DisplayStats`.
   - Display real-time completion percentages and progress toward goals within the command-line interface.

---

### 3. **Badge System & Milestones 🏆**

   Gamify the coding experience by introducing a badge and milestone system to celebrate user accomplishments.

   **Features:**
   - Award badges for reaching specific streaks, commit counts, and language milestones.
   - Create a personal milestone history, encouraging users to break personal records.
   - Adaptive difficulty that increases goal milestones based on past performance.

   **Implementation:**
   - Create a badge system with customizable icons and settings.
   - Store milestone data to generate user-specific insights and set progressive goals.

---

### 4. **Detailed Language and Commit Analytics 📈**

   Gain insight into code contributions by language and commit detail, making it easy to see where time and effort are spent.

   **Features:**
   - Track language-specific contributions across repositories.
   - Display changes in contribution trends by language over time.
   - Provide a breakdown of commit activity to help users visualize contributions in each language.

   **Implementation:**
   - Display language and commit analytics in a dedicated stats section.
   - Track trends and breakdowns using real-time and historical data from `CommitHistory`.

---

### 5. **Team Collaboration Features 👥**

   Enhance Streakode's functionality for team settings, allowing multiple developers to track and share their progress collaboratively.

   **Features:**
   - **Shared Reports**: Generate team reports that combine contributions across shared repositories.
   - **Team Velocity Tracking**: Track and display each team member’s contributions and velocity on team projects.
   - **Leaderboard**: Motivate team members with a leaderboard that highlights individual contributions.
   - **Group Goals**: Enable teams to set and track shared commit goals for collaborative projects.

   **Implementation:**
   - Introduce `team` configuration with options for members and shared projects.
   - Aggregate team member data in a shared file (e.g., JSON/CSV) to generate combined reports.
   - Add privacy settings to let users control the visibility of their stats in team reports.

---

### 6. **Interactive CLI & Configurable Display Options 💡**

   Make Streakode more interactive and customizable to suit user preferences and workflows.

   **Features:**
   - Interactive mode for exploring stats and switching between detailed views.
   - Configurable display options to adjust table styling, width, and output verbosity.
   - Toggleable insights and data sections, allowing users to prioritize key metrics.

   **Implementation:**
   - Develop an `interactive` command to navigate stats via CLI.
   - Add configuration options for table styles, width, and verbosity in the `.streakodeconfig` file.
   - Enable real-time adjustments to output detail levels for flexible display preferences.
