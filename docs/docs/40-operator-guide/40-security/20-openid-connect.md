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
        viewers:
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

### Using Okta as IDP

While Okta [officially supports OpenID Connect and
PKCE](https://auth0.com/docs/authenticate/identity-providers/enterprise-identity-providers/configure-pkce-claim-mapping-for-oidc),
**Kargo OIDC configuration is currently incompatible with this setup** due to
leaving out the `state` parameter in its PKCE flow (see [`akuity/kargo` issue
#3575](https://github.com/akuity/kargo/issues/3575) and [Pull Request
#2916](https://github.com/akuity/kargo/pull/2916) for more).

:::info
Okta has [dedicated
documentation](https://support.okta.com/help/s/article/oidc-authorization-request-failure-invalid-state)
to the error you would encounter if you attempted such a setup.

**The `state` parameter aims at protecting against cross-site request forgery
(CSRF)** attacks by maintaining a state between the request and the callback,
ensuring the response is from a request the client actually initiated. The
client sends a random `state` value in the authentication request and verifies
the same value is returned in the response.

As per the [Okta documentation on why they enforce its
usage](https://support.okta.com/help/s/article/oidc-authorization-request-failure-invalid-state):
> The OAuth 2.0 specification
> [requires](https://datatracker.ietf.org/doc/html/rfc6749#section-10.12) that
> clients protect their redirect URIs against CSRF by sending a value in the
> authorized request that binds the request to the user-agent's authenticated
> state. Using the state parameter is also a countermeasure to several other
> known attacks, as outlined in [OAuth 2.0 Threat Model and Security
> Considerations](https://tools.ietf.org/html/rfc6819).

As per the [OIDC specification of authentication
requests](https://openid.net/specs/openid-connect-core-1_0.html#AuthRequest),
the `state` parameter is **recommended** and not mandatory:
> RECOMMENDED. Opaque value used to maintain state between the request and the
> callback. Typically, Cross-Site Request Forgery (CSRF, XSRF) mitigation is
> done by cryptographically binding the value of this parameter with a browser
> cookie.
:::

Therefore, if you are looking to authenticate on Kargo through Okta, you will
**have to go with the Dex configuration**. Note that going through Dex still
means doing OIDC authentication, just not with PKCE given it's not necessary.

There are a few specificities to be aware of though:

- **Different callback URL**
    - the callback URL will be `https://<hostname for api server>/dex/callback`
      rather than `https://<hostname for api server>/login`
- **`groups` scope to be requested explicitly**
    - While Kargo's OIDC with PKCE implementation [**adds the `groups` scope**
      to the requested ones even if non-standard](#configuring-kargo), **Dex
      doesn't**: it
      [defaults](https://dexidp.io/docs/connectors/oidc/#configuration) to
      [requesting `profile` and `email`
      only](https://github.com/dexidp/website/blob/main/content/docs/connectors/oidc.md?plain=1#L47),
      even if Kargo requests `groups` to it — _it gets lost in the redirection
      between `https://<hostname of api server>/dex/auth/<dex connect id>` and
      `https://<idp issuer>/oauth2/v1/authorize`_.
    - Therefore, you have to **define the `api.oidc.dex.connectors.<your
      connector>.config.scopes` to include `groups` if you want to use it to
      assign admins, viewers or more**.
- **`email_verified` claim to be ignored**
    - Okta has no usage of emails verification in enrollment process, therefore
      the `email_verified` field isn't sent in its OIDC response's claims,
      which makes Dex fail with `Missing "email_verified" in OIDC claim`.
    - To prevent this, you have to **set
      `api.oidc.dex.connectors.<your_connector>.config.insecureSkipEmailVerified`
      to `true`**.
    - See the [Dex documentation about
      it](https://github.com/dexidp/website/blob/main/content/docs/connectors/oidc.md?plain=1#L54-L57),
      the [OIDC standard claims
      documentation](https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims),
      and [the `dexidp/dex` issue
      #1405](https://github.com/dexidp/dex/issues/1405) for more.
- **Groups to be forcely sent**
    - Groups claims (like the rest of OIDC claims through dex) only refresh
      when the ID token is refreshed, meaning the regular refresh flow doesn't
      update the groups claim.
    - As such **the Dex' `oidc` connector doesn't allow `groups`
  claims by default**.
    - You have to **set
      `api.oidc.dex.connectors.<your_connector>.config.insecureEnableGroups` to
      `true`** if you want to have up-to-date `groups` claims to assign
      permissions.
    - Check the [Dex documentation on _"Authentication Through an OpenID
      Connect Provider"_](https://dexidp.io/docs/connectors/oidc/) and [the
      OIDC connector doc on this
      parameer](https://github.com/dexidp/website/blob/main/content/docs/connectors/oidc.md?plain=1#L59-L64)
      for more.

These result in the following configuration snippet that you can use to authenticate on Kargo through Okta and OIDC:

```yaml
api:
  oidc:
    enabled: true
    admins: # This section and `viewers` remains valid even while using Dex
      claims:
        groups:
          - <SOME-OKTA-GROUP>
    dex:
      enabled: true
      env:
        - name: CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: okta
              key: clientSecret
      connectors:
        - id: okta
          name: Okta
          type: oidc
          config:
            issuer: https://<EXAMPLE>.okta.com/
            clientID: <CLIENT_ID>
            clientSecret: $CLIENT_SECRET
            redirectURI: https://<hostname for api server>/dex/callback
            insecureSkipEmailVerified: true
            insecureEnableGroups: true
            scopes:
              - openid
              - profile
              - email
              - groups
```
