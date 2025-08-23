# v0.3.0
* Added Helm chart
* Added argument `--enable-dockerhub-mirror` to enable docker.io mirroring (default is false)
* Added argument `--dockerhub-mirror` to specify `docker.io` mirror to use (default is `mirror.gcr.io`)
* Added `identifier` field to image report(s).  This is digest if it was referenced by digest, otherwise it will be the tag.
* Removed report format `simplified-json` (was not actually simplified at all)
* Fixed error message when no latest semver digest is found (now only logs a debug message)
* Formatting Kubernetes client-side throttling logs more nicely

# v0.2.0
* Major refactor of code and command line arguments

# v0.1.0
* Initial release
