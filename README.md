# watchspatch

`watchspatch` stands for `watch` and `dispatch` and is a filewatcher for moderately complex build workflows.

Think of a project where you'd have a couple frontend javascript files that may require being built, a couple go services etc... `watchspatch` allows you to watch the entire directory and subdirectory tree and depending on which file was modified, trigger a specific command.

I've found it to be a great companion to a make file when I have a couple different items to build (javascript, go, php all in one project), and some local docker-compose container to restart after build.

## usage

Create a `.watchspatch.toml` at your project root. Here's an example:

```toml
version=1 # no choice for now it's always 1


# sometimes your editor might save a file twice in a row or save then `touch` it, triggering more events than wanted.
# `debounce` sets a global latency so that any pattern that was triggered can't be retriggered within `debounce` time of the previous event.
debounce="200ms"


# This is the most interesting part. You define glob patterns and if the file that got changed matches
# a given glob pattern, the corresponding command will be triggered
[patterns]
"*.go"={cmd = "echo toplevel"}
"*/*.js"={cmd = "echo sublevel"}
# Some commands will benefit from a custom debounce value (slow ones in general are a good fit here)
"**/*.php"={cmd = "echo any depth", debounce="1s"}
"go.sum"={cmd="echo gosum:::"}
```

Then run `watchspatch` from this directory. From now on any file event in that directory tree that matches one of the defined glob patterns will trigger a command.
