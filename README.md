# go-ls A expasion on the ls and ls-go imlementation

![ls-go](./img/ls-go.png)

A high performance version of `ls` written in [Go](https://golang.org/).

Build to work with big dictories, mainly to help compare entire file systems,
helping to validate backups or deep copies.

Add some functions over the original ls

- Save to dababase function
- Go template
- Hash files and dirs
- Async Print
- Parallel implementation (See the benchmark)

## Usage

``` sh
usage: go-ls [<flags>] [<paths>...]

Usage: main [<path> ...]

Arguments:
  [<path> ...]    Paths to list.

Flags:
  -h, --help                                       Show context-sensitive help.
  -a, --all                                        Show hidden files;
  -A, --almost-all                                 do not list implied . and ..;
      --author                                     with -l, print the author of each file;
  -b, --escape                                     print C-style escapes for nongraphic characters;
      --block-size=STRING                          with -l, scale sizes by SIZE when printing them; e.g., '--block-size=M'; see SIZE format below;
  -B, --ignore-backups=STRING                      do not list implied entries ending with ~;
  -c, --ctime                                      with -lt: sort by, and show, ctime (time of last modification of file status information); with -l: show ctime and sort by name; otherwise: sort by ctime, newest first;
  -C, --columns                                    list entries by columns;
      --color="always"                             colorize the output; WHEN can be 'always' (default if omitted), 'auto', or 'never';
  -d, --directory                                  list directories themselves, not their contents;
  -D, --dired                                      generate output designed for Emacs\' dired mode;
  -f, --not-sort                                   do not sort, enable -aU, disable -ls --color;
  -F, --classify                                   append indicator (one of */=>@|) to entrie;
      --file-type                                  likewise, except do not append '*'
      --format="long"                              across -x, commas -m, horizontal -x, long -l, single-column -1, verbose -l, vertical -C
      --full-time                                  like -l --time-style=full-iso
  -g, --hide-owner                                 like -l, but do not list owner
      --group-directories-first                    group directories before files; can be augmented with a --sort option, but any use of --sort=none (-U) disables grouping
  -G, --no-group                                   in a long listing, don\'t print group names
  -h, --human-readable                             with -l and -s, print sizes like 1K 234M 2G etc.
      --si                                         likewise, but use powers of 1000 not 1024
  -H, --dereference-command-line                   follow symbolic links listed on the command line
      --dereference-command-line-symlink-to-dir    follow each command line symbolic link that points to a directory
      --hide=STRING                                do not list implied entries matching shell PATTERN (overridden by -a or -A)
      --hyperlink="never"                          hyperlink file names; WHEN can be 'always' (default if omitted), 'auto', or 'never'
      --indicator-style=STRING                     append indicator with style WORD to entry names: none (default), slash (-p),file-type (--file-type), classify (-F)
  -i, --inode                                      print the index number of each file
  -I, --ignore=STRING                              do not list implied entries matching shell PATTERN
  -k, --kibibytes                                  default to 1024-byte blocks for disk usage; used only with -s and per directory totals
  -l, --list                                       use a long listing format
  -L, --dereference                                when showing file information for a symbolic link, show information for the file the link references rather than for the link itself
  -m, --comma-separated                            fill width with a comma separated list of entries
  -n, --numeric-uid-gid                            like -l, but list numeric user and group IDs
  -N, --literal                                    print entry names without quoting
  -o, --no-group-info                              like -l, but do not list group information
  -p, --indicator-style-flag                       append / indicator to directories
  -q, --hide-control-chars                         print ? instead of nongraphic characters
      --show-control-chars                         show nongraphic characters as-is (the default,unless program is 'ls' and output is a terminal)
  -Q, --quote-name                                 enclose entry names in double quotes
      --quote-style="literal"                      use quoting style WORD for entry names: literal, locale, shell, shell-always, shell-escape, shell-escape-always, c, escape (overrides QUOTING_STYLE environment variable)
  -r, --reverse                                    reverse order while sorting
  -R, --recursive                                  recursive list subdirectories recursively;
  -s, --size                                       print the allocated size of each file, in blocks;
  -S, --sort-by-size                               sort by file size, largest first;
      --sort="none"                                sort by WORD instead of name: none (-U), size (-S),time (-t), version (-v), extension (-X);
      --time="creation"                            change the default of using modification times; access time (-u): atime, access, use; change time (-c): ctime, status birth time: birth, creation; with -l, WORD determines which time to show; with --sort=time, sort by WORD (newest first)
      --time-style=STRING                          time/date format with -l; see TIME_STYLE below;
  -t, --sort-by-time                               sort by time, newest first; see --time;
  -T, --tabsize=INT                                assume tab stops at each COLS instead of 8;
  -u, --sort-by-access-time                        with -lt: sort by, and show, access time; with -l: show access time and sort by name; otherwise: sort by access time, newest first;
  -U, --directory-sort                             do not sort; list entries in directory order;
  -v, --natural-sort                               natural sort of (version) numbers within text;
  -w, --width=0                                    set output width to COLS. 0 means no limit;
  -x, --simple-output                              list entries by lines instead of by columns;
  -X, --sort-by-extension                          sort alphabetically by entry extension;
  -Z, --context                                    print any security context of each file;
      --version                                    output version information and exit;
      --template=STRING                            output format using go-template, cannot be used with -o=databse
      --db-connection=STRING                       Db connection string.
      --db-type=STRING                             Db type.

Args:
  [<paths>]  the files(s) and/or folder(s) to display
```

## Install

With `go get`:

```sh
$ go get -u github.com/vitorecomp/go-ls
```

## Contributing

Contributions are muchly appreciated!
Is there another option that would be useful?
Submit a PR!

This repo still a work in progress, so if you want to help let me know.


## Thanks to


