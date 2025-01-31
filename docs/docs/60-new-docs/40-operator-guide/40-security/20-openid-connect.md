---
sidebar_label: OpenID Connect
---

# Authentication with OpenID Connect

By design, Kargo has no built-in system of registering and managing users.
Assuming the lone, built-in admin account is disabled, as recommended
[here](./10-secure-configuration.md#disabling-the-admin-account), you will need
to configure Kargo to authenticate users via an external identity provider
(IDP). This is recommended even in the case of the admin account not having been
disabled, as it is the only way to grant users access to Kargo without
disseminating the admin account's credentials. Conveniently, integrating Kargo
with the IDP already used by your organization has the added benefit of
enabling single sign-on (SSO) for your users.

This document is a comprehensive guide for operators integrating Kargo with an
external IDP.

## OpenID Connect and PKCE

Like many modern applications, Kargo uses the
[OpenID Connect protocol](https://openid.net/developers/how-connect-works/)
(OIDC) for integrating with IDPs. It additionally utilizes
[PKCE](https://auth0.com/docs/get-started/authentication-and-authorization-flow/authorization-code-flow-with-pkce)
(Proof Key for Code Exchange) to secure the
[authorization code flow](https://auth0.com/docs/get-started/authentication-and-authorization-flow/authorization-code-flow).

:::info
Briefly, PKCE is an extension to the authorization code flow that does not
require clients to be issued a secret, which they must then protect. This is
useful for "public clients" such as single-page applications (SPAs) and mobile
apps, which cannot reliably protect a secret from intrepid users.

Historically, public clients have either utilized the less secure implicit flow
or used their own API server (which _can_ protect a client secret) as a
middleman. With PKCE, a client can securely interact with an IDP directly.
:::

Many IDPs support both OIDC and PKCE and Kargo can be integrated directly with
those that do. Consult your identity provider's documentation to discover
whether it supports _both_ of these standards. _If it does not, read on
regardless. We will show how to work around such a constraint._

:::info
Whether you're installing Kargo
[using Helm](../20-advanced-installation/10-advanced-with-helm.md) or
[via Argo CD](../20-advanced-installation/20-advanced-with-argocd.md), the
next two sections assume familiarity with procedures for configuring that
installation.
:::

## Configuration

### Registering Kargo as a Client

To enable integration with your IDP, you will first need to register Kargo as a
client. Carefully consult your IDP's documentation for instructions on how to do
so.

The callback URLs you will need to register are:

- `https://<hostname for api server>/login` (for the UI)
- `https://localhost/auth/callback` (for the CLI)

:::info
If your IDP does not permit you to register multiple callback URLs, you
may need to register two clients -- one each for the UI and the CLI.
:::

__When registration is complete, make note of the issuer URL and client ID
provided by your IDP.__

### Configuring Kargo

When installing Kargo with Helm, all options related to OIDC are grouped under
`api.oidc`.

1. Set `api.oidc.enabled` to `true`.

1. Only if your IDP supports _both_ OIDC and PKCE:

    1. Set `api.oidc.issuerURL` to the issuer URL provided by your IDP.

    1. Set `api.oidc.clientID` to the client ID provided by your IDP.

        If you needed to register two separate clients, use the client ID
        associated with the UI.

    1. Set `api.oidc.cliClientID` to the client secret provided by your IDP.

        If you needed to register two separate clients, use the client ID
        associated with the CLI.

    1. Ensure `api.oidc.dex.enabled` remains set to its default value of
       `false`.

    Example:

    ```yaml
    api:
      oidc:
        enabled: true
        issuerURL: <issuer url>
        clientID: <ui client id>
        cliClientID: <cli client id>
        dex:
          enabled: false
    ```

1. Configure `api.oidc.additionalScopes`:

    This is a list of additional and possibly non-standard scopes that Kargo
    will request from the IDP. For the most part, they map directly to a claim
    you are requesting to be included in ID tokens issued by the IDP.

    By default the `additionalScopes` list contains `groups`, which is a
    non-standard scope/claim, but one that is widely supported. If your IDP does
    not support it, remove that scope from the list.

    If there are additional claims you need because either you or administrators
    of individual projects will use them in mapping users to roles
    (see [Access Controls](30-access-controls.md)), add the
    corresponding scopes to the list. Consult your IDP's documentation to
    discover what scopes are available.

    Example:

    ```yaml
    api:
      oidc:
        # ... omitted for brevity ...
        additionalScopes:
        - groups
        - <additional scope>
        - <another additional scope>
    ```

1. Configure `api.oidc.admins` and `api.oidc.viewers`:

   These map claims in ID tokens to _system-wide_ admin and viewer roles
   respectively. If, for example, every user in the group `devops` should be an
   admin, and every user in the group `developers` should be a viewer, you would
   set these accordingly:

    ```yaml
    api:
      oidc:
        # ... omitted for brevity ...
        admins:
          claims:
            groups:
            - devops
        viewer:
          claims:
            groups:
            - developers
    ```

    :::caution
    Most assignments of users to roles is accomplished at the _project level_
    since individual users' permissions are likely to vary from project to
    project. `api.oidc.admins` and `api.oidc.viewers` are strictly for mapping
    users to _system-wide_ roles.
    :::

    :::note
    It is common to map _all_ authenticated users to the `kargo-viewer`
    `ServiceAccount` to effect broad read-only permissions. These permissions
    _do not_ extend to credentials and other project `Secret`s.
    :::

### Adapting Incompatible IDPs

:::note
If your IDP supports both OIDC and PKCE, you can skip this section entirely.
:::

So your IDP does not support both OIDC and PKCE? In most cases, Kargo can work
around this limitation quite easily through its optional, but seamless
integration with [Dex](https://dexidp.io/), which supports both standards and
can easily be configured to act as a middleman between Kargo and most IDPs.

To configure Kargo to use Dex, set:

1. `api.oidc.dex.enabled` to `true`.

1. Configure one or more [connectors](https://dexidp.io/docs/connectors/) under
   `api.oidc.dex.connectors`.

    But default, no connectors are configured, although the Kargo chart's
    `values.yaml` file includes a few examples, however, Dex's own documentation
    should be counted as the definitive source of information on how to
    configure each available connector.

    To illustrate here, we will used the
    [GitHub connector](https://dexidp.io/docs/connectors/github/). GitHub does
    _not_ support OIDC. By introducing Dex, with proper configuration, as a
    middleman, we can still integrate Kargo with GitHub regardless.

    ```yaml
    api:
      oidc:
        enabled: true
        # ... omitted for brevity ...
        dex:
          enabled: true
          # Adapted from: https://dexidp.io/docs/connectors/github/
          connectors:
          - type: github
            id: github
            name: GitHub
            config:
              clientID: <github client id>
              clientSecret: $CLIENT_SECRET # Best not to include secrets in your values.yaml
              redirectURI: https://<hostname for api server>/dex/callback
              orgs: # Limit access to users in specific orgs; optional but recommended
              - name: <org name>
                teams: # Limit access to users in specific teams; also optional
                - <team name>
                - <another team name>
    ```

    :::info
    The Kargo chart will generate all the remaining Dex configuration for you.
    Only the `connectors` section needs to be provided.
    :::

1. Define environment variables using `api.oidc.dex.env`:

    In the previous step, our connector configuration referenced a
    `$CLIENT_SECRET` environment variable to avoid storing sensitive information
    in a `values.yaml` file.

    To securely provide a value for that environment variable, configure
    `api.oidc.dex.env` like so:

    ```yaml
    api:
      oidc:
        dex:
          # ... omitted for brevity ...
          env:
          - name: CLIENT_SECRET
            valueFrom:
              secretKeyRef:
                name: github-dex
                key: clientSecret
    ```

    The above example would require that you have, through some means, created a
    secret named `github-dex` with a key `clientSecret` within the same
    namespace in which Kargo is installed.
