# Todo

## basic

- (Done)walk over dirs
- (Done)list files
- symbolic links
- use channels
- use golang routines
- make the benchmark builder
- make the node_modules brenchmark

## main

- option -a, --all                  do not ignore entries starting with .
- option -A, --almost-all           do not list implied . and ..
- option --author               with -l, print the author of each file
- option -b, --escape               print C-style escapes for nongraphic characters
- option --block-size=SIZE      with -l, scale sizes by SIZE when printing them; e.g., '--block-size=M'; see SIZE format below
- option -B, --ignore-backups       do not list implied entries ending with ~
- option -c                         with -lt: sort by, and show, ctime (time of last
- option -C                         list entries by columns
- option --color[=WHEN]         colorize the output; WHEN can be 'always' (defaultif omitted), 'auto', or 'never'; more info below
- option -d, --directory            list directories themselves, not their contents
- option -D, --dired                generate output designed for Emacs' dired mode
- option-f                         do not sort, enable -aU, disable -ls --color
- option-F, --classify             append indicator (one of */=>@|) to entries
- option --file-type            likewise, except do not append '*'
- option --format=WORD          across -x, commas -m, horizontal -x, long -l, single-column -1, verbose -l, vertical -C
- option --full-time            like -l --time-style=full-iso
- option -g                         like -l, but do not list owner
- option --group-directories-first group directories before files; can be augmented with a --sort option, but any use of --sort=none (-U) disables grouping
- option -G, --no-group             in a long listing, don't print group names
- option -h, --human-readable       with -l and -s, print sizes like 1K 234M 2G etc.
- option --si                   likewise, but use powers of 1000 not 1024
- option -H, --dereference-command-line follow symbolic links listed on the command line
- option --dereference-command-line-symlink-to-dir follow each command line symbolic link that points to a directory
- option --hide=PATTERN         do not list implied entries matching shell PATTERN (overridden by -a or -A)
- option --hyperlink[=WHEN]     hyperlink file names; WHEN can be 'always' (default if omitted), 'auto', or 'never'
- option --indicator-style=WORD  append indicator with style WORD to entry names: none (default), slash (-p),file-type (--file-type), classify (-F)
- option -i, --inode                print the index number of each file
- option -I, --ignore=PATTERN       do not list implied entries matching shell PATTERN
- option -k, --kibibytes            default to 1024-byte blocks for disk usage;
                               used only with -s and per directory totals
- (DONE) option -l                         use a long listing format
- option -L, --dereference          when showing file information for a symbolic
                               link, show information for the file the link
                               references rather than for the link itself
- option -m                         fill width with a comma separated list of entries
- option -n, --numeric-uid-gid      like -l, but list numeric user and group IDs
- option -N, --literal              print entry names without quoting
- option -o                         like -l, but do not list group information
- option -p, --indicator-style=slash
                             append / indicator to directories
- option -q, --hide-control-chars   print ? instead of nongraphic characters
- option --show-control-chars   show nongraphic characters as-is (the default,
                               unless program is 'ls' and output is a terminal)
- option -Q, --quote-name           enclose entry names in double quotes
- option --quoting-style=WORD   use quoting style WORD for entry names:
                               literal, locale, shell, shell-always,
                               shell-escape, shell-escape-always, c, escape
                               (overrides QUOTING_STYLE environment variable)
- option -r, --reverse              reverse order while sorting
- option -R, --recursive            list subdirectories recursively
- option -s, --size                 print the allocated size of each file, in blocks
- option -S                         sort by file size, largest first
- option --sort=WORD            sort by WORD instead of name: none (-U), size (-S),
                               time (-t), version (-v), extension (-X)
- option --time=WORD            change the default of using modification times;
                               access time (-u): atime, access, use;
                               change time (-c): ctime, status;
                               birth time: birth, creation;
                             with -l, WORD determines which time to show;
                             with --sort=time, sort by WORD (newest first)
- option --time-style=TIME_STYLE  time/date format with -l; see TIME_STYLE below
- option -t                         sort by time, newest first; see --time
- option -T, --tabsize=COLS         assume tab stops at each COLS instead of 8
- option -u                         with -lt: sort by, and show, access time;
                               with -l: show access time and sort by name;
                               otherwise: sort by access time, newest first
- option -U                         do not sort; list entries in directory order
- option -v                         natural sort of (version) numbers within text
- option -w, --width=COLS           set output width to COLS.  0 means no limit
- option -x                         list entries by lines instead of by columns
- option -X                         sort alphabetically by entry extension
- option -Z, --context              print any security context of each file
- option -1                         list one file per line.  Avoid '\n' with -q or -b
      --help     display this help and exit
      --version  output version information and exit

## main add funtions

- implement the golang output formater
- implement the hash of files
- implement the comparable sort
- implement the fast mode (incremental)

## Database

- implement the realtional database connection
- add the paramenters
- create table of dirs
- save the registries

## File

- implement the plain text save
- implement the json format
