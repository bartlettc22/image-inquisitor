# Development Guide

## Image References
Images references can take two forms:
```
[registry]/[repository]:[tag]
```
or
```
[registry]/[repository]@[digest]
```

### Tagged Digest References
A tagged digest referece refers to a image reference that includes both a tag and a digest. This takes the form:
```
[registry]/[repository]:[tag]@[digest]
```
While not technically a valid image reference, this can be observed with certain Kubernetes deployments and can be useful to describe images with tags that are mutable and may change over time. With the inclusion of the digest, it indicates a more specific, immutable image reference while maintaining the original tag reference.

For all intents and purposes, the tag portion of a tagged digest reference is ignored and the digest is used for any image operations.

### Naming Expansion
Images hosted on Docker Hub can be referenced by shorthand names (i.e. `nginx:latest`).  These image references will be automatically expanded to their full indentifier. The following are equivilant valid image names:

|Image Reference|Expanded Image Reference|
|-|-|
|`nginx`|`index.docker.io/library/nginx:latest`|
|`nginx:latest`|`index.docker.io/library/nginx:latest`|
|`library/nginx:latest`|`index.docker.io/library/nginx:latest`|
|`index.docker.io/library/nginx:latest`|`index.docker.io/library/nginx:latest`|

In addition, image names without a tag or digest will be expanded to include the `latest` tag.

### Term Glossary
Given the example image references below, the following terms are used to describe each parts of the reference:
```
index.docker.io/library/nginx:1.29.0
index.docker.io/library/nginx@sha256:84ec966e61a8c7846f509da7eb081c55c1d56817448728924a87ab32f12a72fb
```

|Name|Example(s)|
|---|---|
|`Reference`|`index.docker.io/library/nginx:1.29.0` or `index.docker.io/library/nginx@sha256:84ec966e61a8c7846f509da7eb081c55c1d56817448728924a87ab32f12a72fb`|
|`ReferencePrefix`|`index.docker.io/library/nginx`|
|`Registry`|`index.docker.io`|
|`Repository`|`library/nginx`|
|`RepositoryNamespace`|`library/`|
|`Tag`|`v1.29.0`|
|`Digest`|`sha:0a0704ac83aa7b896a68b710c24f9502bbae87d32e6c7ef6148ce6a60ce05260`|

## TODO List
- Handle Trivy directory better
    - Add flag to specify directory
    - Add flag to enable cleanup of image scan results
- Feed sources into inventory as they are discovered (i.e. from Kubernetes) to improve speed
- Add support for ignoring Kubernetes resources by label/annotation (i.e. ignoring GKE/EKS-managed resources)
- Add support for running in daemon mode
- Add Cache for docker.io registry
- Add support for private registries
- Scoring (weighting) of image priorities
