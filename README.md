<div align="center" width="100%">

<a href="https://github.com/gravitton">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/gravitton/blog/refs/heads/main/docs/images/logo-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/gravitton/blog/refs/heads/main/docs/images/logo-light.svg">
  <img alt="Gravitton blog" src="https://raw.githubusercontent.com/gravitton/blog/refs/heads/main/docs/images/logo-light.svg" width="300">
</picture>
</a>

[![Build Status][ico-workflow]][link-workflow]
[![Website][ico-website]][link-website]

Posts on the decisions behind the packages and the engine

<hr>

</div>

The source of [the blog][link-website], built with [Hugo][link-hugo] and deployed by GitHub Pages.

## Writing

```shell
hugo server --buildDrafts
```

A post is a Markdown file in `content/posts`, or a folder `content/posts/<post>/index.md` when it has images. It stays
out of the published site while its front matter says `draft: true`.

### Images

The images of a post sit next to its `index.md`.

```markdown
![Alt text](shot@2x.png "Optional caption, Markdown allowed")
```

- An image on its own line becomes a framed figure; the title is its caption.
- `shot@2x-dark.png` next to `shot@2x.png` is served to readers in dark mode.
- `@2x` in the name marks a retina capture, shown at half its pixel size.
- `{.plain}` on the line after the image drops the frame, for diagrams and transparent images.

## Code behind the posts

Every number and snippet in a post comes from a Go module in `code/<post>`, pinned to a published tag of the package
it describes. CI runs each one before building the site, so a post cannot quote output the code no longer produces.

```shell
cd code/<post> && go run .
```

## Publishing

Pushing to `main` builds and deploys. In the repository settings, Pages must use **GitHub Actions** as its source.


[ico-workflow]:             https://img.shields.io/github/actions/workflow/status/gravitton/blog/pages.yml?branch=main&style=flat-square
[ico-website]:              https://img.shields.io/website?url=https%3A%2F%2Fgravitton.github.io%2Fblog%2F&style=flat-square&up_color=e8883a

[link-workflow]:            https://github.com/gravitton/blog/actions
[link-website]:             https://gravitton.github.io/blog/
[link-hugo]:                https://gohugo.io
