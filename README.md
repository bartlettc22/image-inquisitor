# Image Inquisitor

<big>Creates a report combining CVE data (using Trivy) with lastest image information.  Meant to be used by Loki to produce a dashboard to make it easier to prioritize upgrades</big>

<img src="image-inquisitor.png " alt="drawing" width="500" />

## Image References
Images references take the form:
```
[registry]/[repository][:tag|@digest]
```

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

### Fully Qualified Image Reference
A fully qualified image reference (FQIR) refers to a image reference that includes both a tag and a digest. This takes the form:
```
[registry]/[repository]:[tag]@[digest]
```
While not a valid image reference as it includes both a tag and a digest, it can be useful to describe images with tags that are mutable and may change over time. With the inclusion of the digest, it indicates a more specific, immutable image reference while maintaining the original tag reference.

## Sources
- file
- kubernetes
- registry
- registryLatestSemver (internal)