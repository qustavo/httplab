# HTTPLab [![Build Status](https://travis-ci.org/gchaincl/httplab.svg?branch=master)](https://travis-ci.org/gchaincl/httplab)
An interactive web server.

HTTPLabs let you inspect HTTP requests and forge responses.

[![asciicast](https://asciinema.org/a/c613qjyikodunp72ox54irn2j.png)](https://asciinema.org/a/c613qjyikodunp72ox54irn2j)

# Install
### Golang
```bash
go get github.com/gchaincl/httplab
```

### Archlinux
```
yaourt httplab
```

### Binary distribution
Each release provides pre-built binaries for different architectures, you can download them here: https://github.com/gchaincl/httplab/releases/latest

## Help
```
Usage of httplab:
  -config string
        Specifies custom config path.
  -port int
        Specifies the port where HTTPLab will bind to. (default 10080)
  -version
        Prints current version.
```

### Key Bindings
Key                                     | Description
----------------------------------------|---------------------------------------
<kbd>Tab</kbd>                          | Next Input
<kbd>Shift+Tab</kbd>                    | Previous Input
<kbd>Ctrl+a</kbd>                       | Apply Response changes
<kbd>Ctrl+s</kbd>                       | Save Response as
<kbd>Ctrl+l</kbd>                       | Toggle responses list
<kbd>Ctrl+o</kbd>                       | Open Body file
<kbd>Ctrl+b</kbd>                       | Switch Body mode
<kbd>Ctrl+h</kbd>                       | Toggle Help
<kbd>q</kbd>                            | Close popup
<kbd>PgUp</kbd>                         | Previous Request
<kbd>PgDown</kbd>                       | Next Request
<kbd>Ctrl+c</kbd>                       | Quit

HTTPLab uses file to store pre-built responses, it will look for a file called `.httplab` on the current directory if not found it will fallback to `$HOME`.
A sample file can be found [here](https://github.com/gchaincl/httplab/blob/master/.httplab.sample).

_HTTPLab is heavily inspired by [wuzz](https://github.com/asciimoo/wuzz)_
