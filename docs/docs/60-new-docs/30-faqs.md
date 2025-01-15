---
sidebar_label: FAQs
---

# Frequently Asked Questions

Welcome to the FAQ page. Here you'll find answers to some of the most common
questions about Kargo.

## General Questions

### _What is Kargo?_

Kargo is an unopinionated
[continuous promotion](#what-exactly-is-continuous-promotion-anyway) platform
that helps developers orchestrate the movement of new code and configuration
through the various "stages" of their applications' lifecycles using GitOps
principles.

### _What exactly is "continuous promotion," anyway?_

If you have some familiarity with GitOps, you know that _GitOps agents_ like
[Argo CD](https://argoproj.github.io/cd/) excel at making the actual state of a
Kubernetes cluster reflect _desired state_ that is expressed declaratively and
stored in a Git repository.

GitOps provides no specific guidance, however, on how desirable changes can be
propagated from the desired state of one stage in an application's lifecycle to
the desired state of the next. Any means by which such a thing might be
accomplished, we would consider to be a "promotion process." When that process
is codified into a pipeline and either partly or fully automated, we consider it
"continuous promotion."

**_Follow up question: Why is this not "continuous deployment?"_**

Because the processes we're focused on do not _perform_ deployments. They focus
on propagating changes to desired state so that a GitOps agent like Argo CD can
then perform the heavy lifting.

### _So “stage” is just another word for “environment?”_

Not exactly, but you could think of it that way if it suits your use case.

Technically, a stage is a _promotion target_. It represents a certain set of
desired state that needs to be altered by a promotion process. The underlying
resources that a GitOps agent will reconcile against that desired state can be
varied according to your needs. It could be a particular instance of your entire
application or just a few microservices that are part of a larger whole. It
could even be an entire Kubernetes cluster if that's what fits your use case.

### _Is Kargo open source?_

Yes it is! You can find the project [on GitHub](https://github.com/akuity/kargo).

If you like what we're doing, please give us a 🌟!

### _How do I get started?_

These very docs are a great place to start. In particular, we recommend checking
out the [Core Concepts](./60-user-guide/10-core-concepts.md) section or, if you're ready to
get your hands dirty, our [Quickstart](./20-quickstart.md).

### _Where can I get support?_

Project maintainers as well as the broader Kargo community are usually quite
responsive to [issues](https://github.com/akuity/kargo/issues) and
[discussions](https://github.com/akuity/kargo/discussions) in the GitHub
repository.

Our community [Discord channel](https://akuity.community) is also quite active
and you're invited to join us there!

If you're interested in a commercial distribution of Kargo or professional
support, check out [akuity.io](https://akuity.io).

### _How can I contribute to the project?_

Find us [on GitHub](https://github.com/akuity/kargo). Open issues. Ask
questions... or even _answer_ questions!

If you're interested in contributing code, our
[Contributor Guide](./50-contributor-guide/index.md) will help you get started. You'll also
find a lot of open issues labeled as
[good first issue](https://github.com/akuity/kargo/labels/good%20first%20issue)
or [help wanted](https://github.com/akuity/kargo/labels/help-wanted). If you
want to work on any of these, comment on the issue to let us know, so we can
assign it to you to help prevent duplicated efforts. If work or life gets in the
way and you can't complete the issue -- no problem. Just let us know.

If you're interested in contributing an entire, new feature, please propose the
feature first and discuss with maintainers before putting a lot of effort into
the implementation.

## Technical Questions

### _Does Kargo force me to work with a separate branch per stage?_

No, it doesn't, although it's a common misconception that it does.

Fundamentally, Kargo needs a place to _store_ the output of your promotion
processes so that it can be picked up and applied by a GitOps agent like Argo
CD. For all intents and purposes, this may as well be an S3 bucket, but as
the term "GitOps agent" implies, the output of those processes will be most
accessible to those agents if it is stored in a Git repository.

Storing the output of your promotion processes in stage-specific branches is a
practice that's been unfairly maligned through misunderstanding of a certain
infamous blog post, which was actually asserting that _GitFlow_ has no place in
GitOps.

Leveraging stage-specific branches is a practice that we do in fact encourage,
but it is by no means a requirement. It is equally tenable to store the output
of promotion processes within a well-thought-out directory structure within a
single branch -- even your `main` branch.

### _Does Kargo support monorepos?_

We get this question _a lot._ In fact, it would seem the majority of our users
are working with monorepos. The short answer _yes._

The longer answer is that Kargo is unopinionated about whether you use one
repository or many. It's also mostly unopinionated about how you structure those
repositories, but it _is_ important that you segregate the configurations for
individual applications or services such that commits to your repository can
easily be selected or ignored on the basis of what paths they affect.

Our [Patterns](./60-user-guide/30-patterns.md) section will provide suggestions for how
to structure monorepos to enable various scenarios.

### _Does Kargo support microservices?_

Yes it does. And there are a lot of different ways Kargo can support you,
depending on your specific needs.

### _What if I need to promote several microservices as a unit?_

In an ideal world, the lifecycles of all microservice are completely independent
of one another. But we don't live in an ideal world. Sometimes you need to
ensure that state changes for a number of related microservices are promoted
together as a unit. There are a number of different ways to achieve this with
Kargo, depending on your specific needs.

Our [Patterns](./60-user-guide/30-patterns.md) section provides additional guidance on
this topic.

**_Follow up question: What if I need to promote several microservices in a
specific order?_**

Kargo can accommodate this as well, and once again there are a number of ways
to approach it depending on your needs and our
[Patterns](./60-user-guide/30-patterns.md) section should help.

### _How do I integrate with multiple Argo CD control planes?_

To get an overview of how this can be achieved, head on over to our
[Architecture](./40-operator-guide/30-architecture.md) section to learn about the topology of
a large-scale Kargo deployment.

### _How do I integrate Kargo into my CI pipelines?_

Truthfully, we hope you don't find the need to do this -- at least not
_directly._

The main impetus for developing Kargo was the lack of tools to comprehensively
effect [continuous promotion](#what-exactly-is-continuous-promotion-anyway). In
this vacuum, the tendency we'd observed was for teams to cobble together bespoke
workflows using a variety of scripts and tools. Chief among these tended to be
CI platforms like Jenkins and GitHub Actions, which are excellent at what they
do (testing code and building artifacts quickly and synchronously), but tend to
be poor at managing the asynchronous, distributed, and complex workflows that
are necessary for continuous promotion. These cobbled together workflows tended
to be difficult to understand, maintain, and scale, and seldom provided the
observability that comes with a single, comprehensive tool.

In short, we built Kargo to be a better alternative. We believe your CI system
remains as important as ever, but that its role is to test code and build
artifacts. Kargo's role is to _notice_ new artifacts and move them through the
stages of your application's lifecycle. This means the (indirect) integration
between your CI system and Kargo are your artifact repositories.

**_Follow up question: What if I really need to?_**

It's possible, of course. Please reach out to
[the maintainers or the community](#where-can-i-get-support) to share your use
case and learn about your options. Understanding your needs will help us to
identify possible gaps in Kargo's capabilities.

### _How do I implement SSO?_

Kargo can be configured to authenticate users with any identity provider that
supports [OpenID Connect](https://openid.net/developers/how-connect-works/)
with [PKCE](https://oauth.net/2/pkce/). This includes most major identity
management platforms like Okta, Auth0, and Microsoft Entra ID (formerly Azure
Active Directory).

Through optional and seamless integration with [Dex](https://dexidp.io/), Kargo
can also integrate with a variety of identity providers that either don't
support PKCE or don't support OpenID Connect at all (GitHub, for example).

Refer to our
[OpenID Connect integration docs](./40-operator-guide/40-security/20-openid-connect.md)
for comprehensive coverage of this topic.
