# go.anx.io

This repository holds the templates and configuration for our go vanity URL website at go.anx.io.


## Usage

* add your package to `packages.yaml`, optionally overriding `targetName` and `summary`
* add a GitHub actions workflow to trigger updates when code is pushed to your repository
* be sure your `go.mod` uses the correct import path `go.anx.io/$targetName`
* profit :)

You can look at [anexia-it/go-anxcloud#96](https://github.com/anexia-it/go-anxcloud/pull/96) for an example
what to do, there is also a handy `sed` in the comments to change import paths over the whole repository.

```yaml
# anexia-it/go.anx.io/packages.yaml

# This package can be imported as go.anx.io/awesomeLibrary
- source:     https://github.com/anexia-it/go-awesome-library.git
  targetName: awesomeLibrary
  summary:    This library does some really awesome things

# This package can be imported as go.anx.io/go-boring-library
- source:     https://github.com/anexia-it/go-boring-library.git
```

`targetName` defaults to the last part of the URL without the `.git`, `summary` to the first top-level
header in `README.md` on the default branch.


Add this as a new workflow or add the job `trigger` to one of your existing workflows. You can also modify it
to run after your tests went through. Make sure to run it for both branches and tags.

```yaml
# anexia-it/go-awesome-library/.github/workflows/push.yaml

name: Trigger go.anx.io update
on:
  push:

jobs:
  trigger:
    name:    Trigger go.anx.io update
    runs-on: ubuntu-latest
    steps:
    - uses: anexia-it/go.anx.io@main
      env:
        GOANXIO_E5E_TOKEN: "${{ secrets.GOANXIO_E5E_TOKEN }}"
```


## The update trigger

Triggering workflows in a repository from another repositories workflow needs a personal access token (PATs) to
create a `repository_dispatch` event, which workflows can be triggered on. Since those PATs are powerful and we
want as less config as possible in each library (so not adding this token to every libraries settings), we use
an E5E function to do the actual GitHub API call. This way we have that token securely in our own infrastructure
in a piece of code that can only trigger this specific API.

The E5E function is "System Engineering / github-dispatch / Trigger go.anx.io rebuild" and called via Frontier
"System Engineering / github-dispatch / trigger go.anx.io rebuild".


## Contributing

Contributions are welcome! Read the [Contributing Guide](CONTRIBUTING.md) for more information.

Only packages by Anexia will be published on go.anx.io, though.


## Licensing

See [LICENSE](LICENSE) for more information.
