# HTTPLab
An interactive web server.

HTTPLabs let you inspect HTTP requests and forge responses.

[![asciicast](https://asciinema.org/a/c613qjyikodunp72ox54irn2j.png)](https://asciinema.org/a/c613qjyikodunp72ox54irn2j)

## Install
```bash
go get github.com/gchaincl/httplab
```
## Help
```
Usage of httplab:
  -port int
        Specifies the port where HTTPLab will bind to (default 10080)
```

### Key Bindings
Key                                     | Description
----------------------------------------|---------------------------------------
<kbd>Tab</kbd>                          | Next Input
<kbd>Shift+Tab</kbd>                    | Previous Input
<kbd>Ctrl+a</kbd>                       | Apply Response changes
<kbd>Ctrl+s</kbd>                       | Save Response as
<kbd>Ctrl+l</kbd>                       | Toggle responses list
<kbd>Ctrl+h</kbd>                       | Toggle Help
<kbd>Ctrl+c</kbd>                       | Quit

_HTTPLab is heavily inspired by [wuzz](https://github.com/asciimoo/wuzz)_
