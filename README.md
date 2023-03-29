# golog
 4everland internal go service log format
## Install
 ```go get github.com/4everland/golog```
# Usage
```
 logger := golog.NewFormatStdLogger(
        os.Stdout, 
        golog.WithFilterLevel(log.LevelInfo), 
        golog.WithServerName("name"", "version"))
```
