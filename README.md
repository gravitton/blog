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
hugo server
```

A post is a folder `content/posts/<number>-<slug>`, numbered in the order it was planned: the text in `index.md`, its
images beside it, and its code in `code/`. The front matter gives the `slug`, which keeps the number out of the URL,
and `keywords`, which go into the page's metadata. A post stays out of the published site while its front matter
says `draft: true` or its date is still to come; `hugo server` shows it anyway, through
`config/development/hugo.toml`, marked as a draft.

### Links between posts

A post links back to an earlier one with a relative link, `[the previous post](../<slug>/)`. A mention of a later
post goes through the `post` shortcode, which renders a link once that post is published and plain text until
then, so no post ever links to a page that is not there yet:

```markdown
{{</* post "<slug>" "the next post" */>}}
```

Every post ends with a Previous and a Next link to its neighbours in the series, from the dates, among the posts
published so far.

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

Every number and snippet in a post comes from the Go module in its `code/` folder, pinned to a published tag of the
package it describes. CI runs each one before building the site, so a post cannot quote output the code no longer
produces, and Hugo leaves the folder out of the published site.

```shell
cd content/posts/<post>/code && go run .
```

A draft may be written against a release that is not tagged yet. Its module requires the coming tag, and a
git-ignored `go.work` beside it replaces that module with the sibling checkout for `go run .` meanwhile:

```
go 1.27

use .

replace github.com/gravitton/<package> => ../../../../../<package>
```

CI skips such a module with a warning while the post is a draft and fails on it once the post is published.

## Publishing

Pushing to `main` builds and deploys. In the repository settings, Pages must use **GitHub Actions** as its source.


[ico-workflow]:             https://img.shields.io/github/actions/workflow/status/gravitton/blog/main.yml?branch=main&style=flat-square
[ico-website]:              https://img.shields.io/website?url=https%3A%2F%2Fgravitton.github.io%2Fblog%2F&style=flat-square&up_color=e8883a

[link-workflow]:            https://github.com/gravitton/blog/actions
[link-website]:             https://gravitton.github.io/blog/
[link-hugo]:                https://gohugo.io
