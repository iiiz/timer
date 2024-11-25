## timer

```
usage: timer [command args]
        start    [-at 00:00] [task]      Start tracking time for a task identifier, may be of an upstream task format or unformatted.
        stop                             Stop tracking time.
        cancel                           Cancel tracking time.
        status                           Prints time tracking status.
        log      [-f yyyy-mm-dd]         Print log of the current day or from a specified date.
Advanced usage:
        ps1      Output prompt complication.
        precmd   Check current directory and prompt to start time tracking, for use as zsh precommmand function.
```

#### Config

The default config file is `~/.timer/config` and can be configured for either an upstream jira or gitlab service.

Default Config:

```
billable_enable=no
```

Example gitlab config:

```
upstream_service=gitlab
url=https://gitlab.example.com
token=YOUR_PERSONAL_ACCESS_TOKEN
default_gitlab_project_id=9999999
```

- required scopes `api`
- Default project id (required, will be optional in future): search for matching issue numbers in this project ignoring the current local git repo.
  Useful if you have multiple projects but only one tracking issues across them.
  Project ID can be copied from the three dot menu (top right) of a project home page.

#### zsh prompt and precmd hook example

```sh
timer_hook() {
    timer precmd
    psvar[1]=$(timer ps1)
}

add-zsh-hook precmd timer_hook

PROMPT="%1v ->"
```

To modify your existing prompt template just `echo $PROMPT` and copy to your zshrc and add `%1v` as you'd like it formatted.

Note: `psvar[X]` where X is 1-9 equates to `%Xv`
