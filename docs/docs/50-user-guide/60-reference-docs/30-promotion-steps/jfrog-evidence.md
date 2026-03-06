---
sidebar_label: jfrog-evidence
description: Manages evidence creation, verification, and deletion for artifacts in JFrog Artifactory.
---

# `jfrog-evidence`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info

This promotion step is only available in Kargo on the [Akuity Platform](https://akuity.io/akuity-platform), versions v1.7 and above.

:::

The `jfrog-evidence` promotion step provides comprehensive integration with JFrog Artifactory's evidence management capabilities, allowing you to create, verify, and delete evidence for artifacts. This enables secure attestation and verification of your promotion workflows through cryptographically signed evidence that can track artifact provenance, test results, and compliance status.

This promotion step supports three main operations: creating evidence with digital signatures, querying and verifying existing evidence, and cleaning up evidence when needed. It's particularly valuable for maintaining supply chain security and compliance in enterprise environments.

For more information about JFrog's Evidence feature, see the [official JFrog Evidence documentation](https://jfrog.com/help/r/jfrog-artifactory-documentation/evidence-management).


## Credentials Configuration

All JFrog Evidence operations require proper authentication credentials stored in a Kubernetes `Secret`.

| Name                     | Type     | Required | Description                                                                     |
| ------------------------ | -------- | -------- | ------------------------------------------------------------------------------- |
| `credentials.secretName` | `string` | N        | Name of the `Secret` containing the JFrog credentials in the project namespace. |
| `credentials.sharedSecretName` | `string` | N  | Name of the `Secret` containing the JFrog credentials in the `shared-resources-namespace`. |

:::info

Either `credentials.secretName` or `credentials.sharedSecretName` must be set, but not both.

:::

The referenced `Secret` should contain the following keys:

- `url`: The URL of your JFrog Artifactory instance (e.g., `https://yourcompany.jfrog.io`)
- `accessToken`: Your JFrog access token with appropriate permissions for evidence management
- `privateKey` (optional): Private key in PEM format for signing evidence. This key can be stored in a separate secret and referenced when required.

## Evidence Management

### Create Evidence

Creates cryptographically signed kargo promotion evidence for an artifact in JFrog Artifactory.

#### Configuration

| Name                     | Type     | Required | Description                                                                                                             |
| ------------------------ | -------- | -------- | ----------------------------------------------------------------------------------------------------------------------- |
| `create.privateKey`      | `string` | Y        | The private key to sign the evidence.                                                                                   |
| `create.packageRepo`     | `string` | Y        | The package repository name of the artifact.                                                                            |
| `create.packageName`     | `string` | Y        | The package name of the artifact inside the repository.                                                                 |
| `create.packageVersion`  | `string` | Y        | The package version of the artifact.                                                                                    |
| `create.promotionStatus` | `string` | Y        | Status indicating the promotion state. Valid values: `Pending`, `Running`, `Succeeded`, `Failed`, `Errored`, `Aborted`. |
| `create.privateKeyAlias` | `string` | N        | Name for the public key created from the private key. Used for verification. If omitted, verification is skipped.       |
| `create.metadata`        | `object` | N        | JSON metadata that will be stored inside the evidence predicate.                                                        |

#### Output

| Name   | Type     | Description                                                      |
| ------ | -------- | ---------------------------------------------------------------- |
| `name` | `string` | The name/identifier of the created evidence for later reference. |


The Kargo promotion evidence predicate stored in the artifact will have a predicate type of `https://akuity.io/evidence/promotion/v1` and will look something like this:

```json
{
  "project": "my-project",
  "stage": "production",
  "promotion": "production.01k22r0jkhfgheh57ywp9t15ya.67018d8",
  "actor": "admin@company.com",
  "freightName": "abc123def456",
  "metadata": {
    "env": "production",
    "result": "pass",
    "testSuite": "integration-tests",
    "coverage": "85%"
  },
  "promotionStatus": "Succeeded",
  "promotionTimestamp": 1672531200
}
```


#### Example

This example creates evidence for a successful test promotion with custom metadata.

```yaml
steps:
- uses: jfrog-evidence
  as: create-test-evidence
  config:
    credentials:
      secretName: jfrog-credentials
    create:
      privateKey: ${{ secret("jfrog-credentials").privateKey }}
      privateKeyAlias: "test-evidence-key"
      packageRepo: ${{ vars.packageRepo }}
      packageName: ${{ vars.packageName }}
      packageVersion: ${{ imageFrom(vars.packageRegistry+"/"+vars.packageRepo+"/"+vars.packageName).Tag }}
      metadata:
        env: "${{ ctx.stage }}"
        result: "pass"
        testSuite: "integration-tests"
        coverage: "85%"
      promotionStatus: "Succeeded"
```

### Delete Evidence

Removes specific evidence from an artifact in JFrog Artifactory. This is typically used as a cleanup step in conjunction with the create step when a promotion fails.

#### Configuration

| Name                    | Type     | Required | Description                                             |
| ----------------------- | -------- | -------- | ------------------------------------------------------- |
| `delete.packageRepo`    | `string` | Y        | The package repository name of the artifact.            |
| `delete.packageName`    | `string` | Y        | The package name of the artifact inside the repository. |
| `delete.packageVersion` | `string` | Y        | The package version of the artifact.                    |
| `delete.evidenceName`   | `string` | Y        | Evidence name/identifier to delete.                     |

#### Output

This step does not produce any output.

#### Example

This example deletes evidence when a promotion fails, using the evidence name from a previous step.

```yaml
steps:
- uses: jfrog-evidence
  as: cleanup-failed-evidence
  if: ${{ failure() && status('create-test-evidence') == 'Succeeded' }}
  config:
    credentials:
      secretName: jfrog-credentials
    delete:
      packageRepo: ${{ vars.packageRepo }}
      packageName: ${{ vars.packageName }}
      packageVersion: ${{ imageFrom(vars.packageRegistry+"/"+vars.packageRepo+"/"+vars.packageName).Tag }}
      evidenceName: ${{ outputs['create-test-evidence'].name }}
```

### Process Evidence (Query and Verify)

Queries for existing evidence and verifies its authenticity and content. This operation is commonly used as a promotion gating mechanism to ensure that only artifacts with valid evidence and compliance status proceed through the deployment pipeline. This step can also extract data from the queried evidence as step outputs, which can then be used as inputs in subsequent steps.

#### Configuration

| Name                                | Type      | Required | Description                                                                                                 |
| ----------------------------------- | --------- | -------- | ----------------------------------------------------------------------------------------------------------- |
| `process.query.packageRepo`         | `string`  | Y        | The package repository name of the artifact.                                                                |
| `process.query.packageName`         | `string`  | Y        | The package name of the artifact inside the repository.                                                     |
| `process.query.packageVersion`      | `string`  | Y        | The package version of the artifact.                                                                        |
| `process.query.evidenceNameRegex`   | `string`  | N*       | Regular expression to match evidence names. *Either this or `predicateType` is required.                    |
| `process.query.predicateType`       | `string`  | N*       | The predicate type to query for. *Either this or `evidenceNameRegex` is required.                           |
| `process.verify.verifyExpression`   | `string`  | N**      | Expr expression to evaluate on the predicate. Must return boolean. **One of the verify options is required. |
| `process.verify.localKeys`          | `array`   | N**      | Local private/public keys for signature verification. **One of the verify options is required.              |
| `process.verify.useArtifactoryKeys` | `boolean` | N**      | Use keys present in evidence data for verification. **One of the verify options is required.                |
| `process.outputs`                   | `array`   | N        | Output configurations to extract data from evidence results.                                                |
| `process.outputs[].name`            | `string`  | Y        | Name of the output variable.                                                                                |
| `process.outputs[].expression`      | `string`  | Y        | Expr expression to extract data from evidence query results.                                                |

#### Available Data for Expressions

Both verification expressions (`verifyExpression`) and output expressions (`outputs[].expression`) have access to the following data structure:

```yaml
evidence:
  predicate: <object>        # The evidence predicate data (varies by predicate type)
  predicateType: <string>    # The predicate type of the evidence
  createdAt: <string>        # ISO timestamp when the evidence was created
  name: <string>             # The name/identifier of the evidence
  verified: <boolean>        # Whether the evidence signature was verified
  downloadPath: <string>     # The download path for the evidence
  signingKey:
    alias: <string>          # The alias of the signing key
    publicKey: <string>      # The public key used for signing
```

#### Output

The outputs are dynamic based on the `process.outputs` configuration. Each output creates a variable with the specified name containing the result of the expression evaluation.

#### Example

This example verifies test evidence and extracts relevant information for subsequent steps.

```yaml
steps:
- uses: jfrog-evidence
  as: verify-test-evidence
  config:
    credentials:
      secretName: jfrog-credentials
    process:
      query:
        packageRepo: ${{ vars.packageRepo }}
        packageName: ${{ vars.packageName }}
        packageVersion: ${{ imageFrom(vars.packageRegistry+"/"+vars.packageRepo+"/"+vars.packageName).Tag }}
        predicateType: "https://in-toto.io/attestation/test-result/v0.1"
        evidenceNameRegex: ".*-test-result-.*"
      verify:
        localKeys:
          - ${{ secret("jfrog-credentials").privateKey }}
        useArtifactoryKeys: true
        verifyExpression: |
          evidence.predicate.result == "pass" && 
          evidence.predicate.coverage >= 80
      outputs:
        - name: testResult
          expression: evidence.predicate.result
        - name: coverage
          expression: evidence.predicate.coverage
        - name: createdAt
          expression: evidence.createdAt
        - name: environment
          expression: evidence.predicate.metadata.env

# Use the extracted values in subsequent steps
- uses: http
  config:
    method: POST
    url: https://api.slack.com/api/chat.postMessage
    headers:
    - name: Authorization
      value: "Bearer ${{ secret('slack-token').token }}"
    - name: Content-Type
      value: application/json
    body: |
      ${{ quote({
        "channel": "#deployments",
        "text": "Test evidence verified! Result: " + outputs['verify-test-evidence'].testResult + 
                ", Coverage: " + string(outputs['verify-test-evidence'].coverage) + "%" +
                ", Environment: " + outputs['verify-test-evidence'].environment
      }) }}
```

#### Trusted Release Evidence Verification

JFrog Trusted Release evidence is a special type of evidence generated exclusively for the release stage. This evidence creates a distinct certification artifact that verifies an application version has been officially released with policy evaluation. It is possible to validate this type of evidence with Kargo and use it as a promotion gating mechanism for promotions in production environments.

##### Configuration

This example demonstrates verifying trusted release evidence before production deployment. The `verifyExpression` can be modified according to your requirements.

```yaml
steps:
- uses: jfrog-evidence
  as: verify-trusted-release
  config:
    credentials:
      secretName: jfrog-credentials
    process:
      query:
        packageRepo: ${{ vars.packageRepo }}
        packageName: ${{ vars.packageName }}
        packageVersion: ${{ imageFrom(vars.packageRegistry+"/"+vars.packageRepo+"/"+vars.packageName).Tag }}
        predicateType: "https://jfrog.com/evidence/release_certify/v1"
      verify:
        localKeys:
          - ${{ secret("jfrog-credentials").privateKey }}
        useArtifactoryKeys: true
        verifyExpression: |
          evidence.predicate.release_type == "Trusted Release" && 
          len(evidence.predicate.policy_results) > 0 &&
          evidence.predicate.outcome == "passed"
      outputs:
        - name: releaseType
          expression: evidence.predicate.release_type
        - name: releaseTimestamp
          expression: evidence.predicate.evaluated_at

# Only proceed with production deployment if trusted release is verified in previous step
# Update the Argo CD Application directly. Not ideal for practical purposes.
- uses: argocd-update
  config:
    apps:
      - name: production-app
        namespace: argocd
        sources:
        - repoURL: https://github.com/company/app-config.git
          kustomize:
            images:
            - repoURL: ${{ vars.packageRegistry }}/{{ vars.packageRepo }}/{{ vars.packageName }}
              tag: ${{ imageFrom(vars.packageRegistry+"/"+vars.packageRepo+"/"+vars.packageName).Tag }}
```

This verification ensures that:
- The evidence is for a "Trusted Release"
- Policy evaluations were performed (policy_results is not empty). You can perform more complex checks on the policy_results according to your requirements.
- All compliance checks passed
- The evidence signature is valid

