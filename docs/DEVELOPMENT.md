# Development

This document describes the intricacies of Nobl9 Language Server development workflow.
If you see anything missing, feel free to contribute :)

## Pull requests

[Pull request template](../.github/pull_request_template.md)
is provided when you create new PR.
Section worth noting and getting familiar with is located under
`## Release Notes` header.

## Makefile

Run `make help` to display short description for each target.
The provided Makefile will automatically install dev dependencies if they're
missing and place them under `bin`
(this does not apply to `yarn` managed dependencies).
However, it does not detect if the binary you have is up to date with the
versions declaration located in Makefile.
If you see any discrepancies between CI and your local runs, remove the
binaries from `bin` and let Makefile reinstall them with the latest version.

## CI

Continuous integration pipelines utilize the same Makefile commands which
you run locally. This ensures consistent behavior of the executed checks
and makes local debugging easier.

## Testing

The repository has unit tests which are either granular,
written around specific parts of the program or more general,
which test a running server binary.
The latter are located under [tests](../tests) directory.

## Releases

Refer to [RELEASE.md](./RELEASE.md) for more information on release process.

## Dependencies

Renovate is configured to automatically merge minor and patch updates.
For major versions, which sadly includes GitHub Actions, manual approval
is required.
