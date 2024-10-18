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