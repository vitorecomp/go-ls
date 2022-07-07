package cli

type CLI struct {
	ShowHidden        bool   `help:"Show hidden files;" name:"all" short:"a"`
	AlmostAll         bool   `help:"do not list implied . and ..;"  name:"almost-all" short:"A"`
	Author            bool   `help:"with -l, print the author of each file;"  name:"author"`
	Escape            bool   `help:"print C-style escapes for nongraphic characters;"  name:"escape" short:"b"`
	BlockSize         string `help:"with -l, scale sizes by SIZE when printing them; e.g., '--block-size=M'; see SIZE format below;"  name:"block-size"`
	IgnoreBackups     string `help:"do not list implied entries ending with ~;"  name:"ignore-backups" short:"B"`
	CTime             bool   `help:"with -lt: sort by, and show, ctime (time of last modification of file status information); with -l: show ctime and sort by name; otherwise: sort by ctime, newest first;" short:"c" name:"ctime"`
	Columns           bool   `help:"list entries by columns;" short:"C" name:"columns"`
	Color             string `enum:"always,auto,never" help:"colorize the output; WHEN can be 'always' (default if omitted), 'auto', or 'never';" name:"color" default:"always" `
	Directories       bool   `help:"list directories themselves, not their contents;" name:"directory" short:"d"`
	Dired             bool   `help:"generate output designed for Emacs' dired mode;" name:"dired" short:"D"`
	NotSort           bool   `help:"do not sort, enable -aU, disable -ls --color;" name:"not-sort" short:"f"`
	Classify          bool   `help:"append indicator (one of */=>@|) to entrie;" name:"classify" short:"F"`
	ClassifyFiletypes bool   `help:"likewise, except do not append '*'" name:"file-type"`
	Format            string `enum:"verbose,long,commas, horizontal, across, vertical, single-column" help:"across -x, commas -m, horizontal -x, long -l, single-column -1, verbose -l, vertical -C" name:"format" default:"long"`

	FullTime         bool `help:"like -l --time-style=full-iso" name:"full-time"`
	HideOwner        bool `help:" like -l, but do not list owner" short:"g"`
	GroupDirectories bool `help:"group directories before files; can be augmented with a --sort option, but any use of --sort=none (-U) disables grouping" name:"group-directories-first"`
	NoGroup          bool `help:"in a long listing, don't print group names" name:"no-group" short:"G"`

	HumanReadble bool `help:"with -l and -s, print sizes like 1K 234M 2G etc." name:"human-readable" short:"h"`
	Base10Size   bool `help:"likewise, but use powers of 1000 not 1024" name:"si"`

	FollowLinks    bool   `help:"follow symbolic links listed on the command line" name:"dereference-command-line" short:"H"`
	FollowDirLinks bool   `help:"follow each command line symbolic link that points to a directory" name:"dereference-command-line-symlink-to-dir"`
	Hide           string `help:"do not list implied entries matching shell PATTERN (overridden by -a or -A)" name:"hide"`
	HyperLink      string `enum:"always, auto, never" help:"hyperlink file names; WHEN can be 'always' (default if omitted), 'auto', or 'never'" name:"hyperlink" default:"never"`
	Style          string `help:"append indicator with style WORD to entry names: none (default), slash (-p),file-type (--file-type), classify (-F)" name:"indicator-style"`
	Inodes         bool   `help:"print the index number of each file" name:"inode" short:"i"`
	Ignore         string `help:"do not list implied entries matching shell PATTERN" name:"ignore" short:"I"`
	Base2Size      bool   `help:"default to 1024-byte blocks for disk usage; used only with -s and per directory totals" name:"kibibytes" short:"k"`

	List             bool   `help:" use a long listing format " name:"list" short:"l"`
	ShowLinks        bool   `help:"when showing file information for a symbolic link, show information for the file the link references rather than for the link itself" name:"dereference" short:"L"`
	CommaSeparated   bool   `help:"fill width with a comma separated list of entries" short:"m"`
	NumericUUID      bool   `help:"like -l, but list numeric user and group IDs" name:"numeric-uid-gid" short:"n"`
	Literal          bool   `help:"print entry names without quoting" name:"literal" short:"N"`
	NoGroupInfo      bool   `help:"like -l, but do not list group information" short:"o"`
	IndicatorStyle   bool   `help:" append / indicator to directories" name:"indicator-style" default:"slash" short:"p"`
	HideControlChars bool   `help:"print ? instead of nongraphic characters" name:"hide-control-chars" short:"q"`
	ShowControlChars bool   `help:"show nongraphic characters as-is (the default,unless program is 'ls' and output is a terminal)" name:"show-control-chars"`
	QuoteName        bool   `help:"enclose entry names in double quotes" name:"quote-name" short:"Q"`
	QuoteStyle       string ` enum:"literal, locale, shell, shell-always, shell-escape, shell-escape-always, c, escape" help:"use quoting style WORD for entry names: literal, locale, shell, shell-always, shell-escape, shell-escape-always, c, escape (overrides QUOTING_STYLE environment variable)" name:"quote-style"`
	Reverse          bool   `help:"reverse order while sorting" name:"reverse" short:"r"`

	Recursive  bool   `help:"recursive list subdirectories recursively;" short:"R"`
	Size       bool   `help:"print the allocated size of each file, in blocks;" short:"s" name:"size"`
	SortBySize bool   `help:"sort by file size, largest first;" short:"S"`
	Sort       string `enum:"none, size, time, version, extension" default:"none" help:"sort by WORD instead of name: none (-U), size (-S),time (-t), version (-v), extension (-X);" name:"sort"`
	Time       string `enum:"atime, acess, use, ctime, status, birth, creation" help:"change the default of using modification times; access time (-u): atime, access, use; change time (-c): ctime, status birth time: birth, creation; with -l, WORD determines which time to show; with --sort=time, sort by WORD (newest first)" name:"time"`
	TimeStyle  string `help:"time/date format with -l; see TIME_STYLE below;" name:"time-style"`
	SortByTime bool   `help:"sort by time, newest first; see --time;" short:"t"`
	TabSize    int    `help:"assume tab stops at each COLS instead of 8;" short:"T" name:"tabsize"`

	SortByAccessTime     bool `help:"with -lt: sort by, and show, access time; with -l: show access time and sort by name; otherwise: sort by access time, newest first;" short:"u"`
	DirectorySort        bool `help:"do not sort; list entries in directory order;" short:"U"`
	NaturalSort          bool `help:"natural sort of (version) numbers within text;" short:"v"`
	SetMaxWidthColumns   int  `default:"0" help:"set output width to COLS.  0 means no limit;" short:"w" name:"width"`
	SimpleOutput         bool `help:"list entries by lines instead of by columns;" short:"x"`
	SortByExtension      bool `help:"sort alphabetically by entry extension;" short:"X"`
	PrintSecurityContext bool `help:"print any security context of each file;" short:"Z" name:"context"`

	Version bool `help:"output version information and exit;" name:"version"`

	Template string `help:"output format using go-template, cannot be used with -o=databse "`

	output string `enum:"console,database,file" default:"console" name:"output"`
	DBConn string `help:"Db connection string." name:"db-connection"`
	DBType string `help:"Db connection string." name:"db-type"`

	Paths []string `arg:"" optional:"" name:"path" help:"Paths to list." type:"path" default:"."`
}

var Arguments = CLI{}
