# Release Flow

Version metadata is injected at build time (GoReleaser ldflags) and resolved from module/VCS build info for `go install` and local builds.
Do not hardcode a version string.

```bash
git checkout -b release/vX.Y.Z
git commit --allow-empty -m 'Release vX.Y.Z'
git push origin release/vX.Y.Z
```

merge PR

```bash
git checkout master
git pull origin master
git tag vX.Y.Z
git push origin vX.Y.Z
# run GitHub Action to build packages and make release.
```
