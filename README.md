# goexe

- goexe executes concurrently commands defined in a command-list file. As a default, 20 commands in the command-list file will be concurrently executed. You can control the concurrency by specifing `-cl` option.

- As a default, goexe doesn't show any output produced by each `PASS`ed command: only `FAIL`ed command's output will be shown. If you want to see the output of `PASS`ed commands, then specify `-v` option.

- If `-w` option is specified, the command-list file will be overwritten by a list of commands which is sorted in descending order by elapsed time.
