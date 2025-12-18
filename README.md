# go-ls A expansion on the ls and ls-go implementation

![ls-go](./img/ls-go.png)

A high performance version of `ls` written in [Go](https://golang.org/).

Build to work with big directories, mainly to help compare entire file systems,
helping to validate backups or deep copies.

It's optimized this tasks that it add over the ls

- Save to database function (and used it on a rerun of output with the same id)
- Go template
- Hash/Checksum files and dirs
- Async Print
- Parallel implementation (See the benchmark)

to common tasks it will probably be slow given that the idea is always to collect all the data over a dir/file.

## Usage

``` sh
usage: go-ls [<flags>] [<paths>...]

Usage: main [<path> ...]

Arguments:
  [<path> ...]    Paths to list.

new flags:
    --db-connection=STRING                       Db connection string.
    --db-type=STRING                             Db type (sqlite, Postgres).
    --db-statistics                              Create statics tables, about the dir (if the table is already create it will use it)
    --db-id=STRING                               Id of the database operation, if not provided a new one will be generated.
    --compare                                    output a list of hash, fullpath, size in a csv format that can be used to compare dirs (force sort and checksum parameters)
    --hash                                       calculates the checksum of every file and hash(hash of checksums and other dir hashes) of the files of a dir

ls flags:
  -h, --help                                       Show context-sensitive help.
  -a, --all                                        Show hidden files;
  -R, --recursive                                  recursive list subdirectories recursively;
  --version                                    output version information and exit;

Args:
  [<paths>]  the files(s) and/or folder(s) to display
```

## Install / Running

With the code:

```sh
    go main.go [<flags>] --flags
```

With `go get`:

```sh
    go get -u github.com/vitorecomp/go-ls
```

## Contributing

Contributions are much appreciated!
Is there another option that would be useful?
Submit a PR!

This repo still a work in progress, so if you want to help let me know.

## Thanks to

## Todo

Below is a list of todo tasks for the project

## documentation

- Create the images
- Make the title
- Make the base documentation
- Make the contribution guidelines

## basic

- (Done)walk over dirs
- (Done)list files
- (Done)symbolic links
- (Done)use channels
- implement file output
- implement database output
- add the windows support

## main add functions

- (DONE)implement the hash of files
- (DONE)implement the fast mode (incremental)

## Database

- implement the relational database connection
- add the parameters
- create table of dirs
- save the registries

## File

- on file mode show statistics
- implement the plain text save
- implement the json format

## fix

- scape on the compare full path

## Advanced

- use golang routines
- (DONE)separate the hash in the process
- use the following structure
- -> Walk over folders
- -> Return the list and metadata
- --> list (all files and folders without hash)
- --> metadata (folder tree, pointer to files)
- -> iterate over files
- -> (DONE)add channel on the start to hashing of files
- -> hash the tree when the files are done
- transform it on a lib

## Benchmark

- make the benchmark builder
- make the node_modules benchmark
