# Stuff I Gotta Do Ya Know


## Fixes
- [X] When there is a directory listed to scan, but it doesn't exist, there is a error and no output is generated.
  -> I should add a check to see if the directory exists before trying to scan it.
  -> And just warn the user that the directory doesn't exist or is not accessible.

## Improvements
- [ ] Implement some sort of check for this:
  -> Before just loading data from cache, check if there is a new directory setup in the config/or a directory where we do not have any data.
  -> If there is none just move on to laod from cache and display output normall.
    -> If there is a new directory or a directory where we do not have any data for, then fetch data and update cache.