---
sidebar_label: jira
description: Integrates with Jira to manage issues, comments, and track promotion workflows.
---

# `jira`

:::info Enterprise Feature
This extension is only available for AKP Enterprise Kargo, you can use this feature in Kargo v1.6+ offered on [Akuity Platform](https://akuity.io/akuity-platform).
:::

The `jira` extension provides comprehensive integration with Jira, allowing you to create, update, delete, and search for issues, manage comments, and track promotion workflows. This extension is particularly useful for maintaining traceability between your deployment processes and project management activities.

The extension supports various operations including issue management, comment handling, and status tracking, making it a powerful tool for promotion workflows that require coordination with project management systems.

## Credentials Configuration

All Jira operations require proper authentication credentials stored in a Kubernetes secret.

| Name                     | Type     | Required | Description                                                                                                                |
| ------------------------ | -------- | -------- | -------------------------------------------------------------------------------------------------------------------------- |
| `credentials.secretName` | `string` | Y        | Name of the secret containing the Jira credentials.                                                                        |
| `credentials.namespace`  | `string` | N        | Namespace for the credentials secret. Can be project or cluster secret namespace. If empty, the project namespace is used. |

The referenced secret should contain the following keys:

- `domain`: The domain of your Jira instance or Jira api (e.g., `https://yourcompany.atlassian.net`)
- `username`: Your Jira username or email
- `password`: Your Jira API token or password

:::info Content Formatting
When using `description` or `body` fields, use plain text as Jira does not render markdown. For rich formatting, use the ADF (Atlassian Document Format) alternatives `adfDescription` or `adfBody` instead.
:::

## Issue Management

### Create Issue

Creates a new Jira issue with specified details.

#### Configuration

| Name                         | Type     | Required | Description                                                                                                 |
| ---------------------------- | -------- | -------- | ----------------------------------------------------------------------------------------------------------- |
| `createIssue.projectKey`     | `string` | Y        | The key of the Jira project where the issue will be created.                                                |
| `createIssue.summary`        | `string` | Y        | The summary or title of the issue.                                                                          |
| `createIssue.description`    | `string` | N        | The description of the issue. Supports markdown formatting.                                                 |
| `createIssue.adfDescription` | `object` | N        | ADF (Atlassian Document Format) content for complex formatting. Alternative to `description`.               |
| `createIssue.issueType`      | `string` | N        | The type of issue to create (e.g., 'Bug', 'Task', 'Story').                                                 |
| `createIssue.assigneeEmail`  | `string` | N        | Email of the user to assign the issue to.                                                                   |
| `createIssue.labels`         | `array`  | N        | Labels to add to the issue for categorization.                                                              |
| `createIssue.customFields`   | `object` | N        | Custom fields to set. Keys should match Jira custom field IDs.                                              |
| `createIssue.issueAlias`     | `string` | N        | Override for the freight metadata key used to reference the created issue id. Defaults to `jira-issue-key`. |

#### Output

| Name  | Type     | Description                                           |
| ----- | -------- | ----------------------------------------------------- |
| `key` | `string` | The key/id of the created Jira issue (e.g., EXT-123). |

#### Example

This example creates a new Jira issue to track a deployment, assigns it to a team member, and adds relevant labels.

```yaml
steps:
  - uses: jira
    as: create-deployment-issue
    config:
      credentials:
        secretName: jira-credentials
      createIssue:
        projectKey: DEPLOY
        summary: "Deploy ${{ imageFrom(vars.imageRepo).Tag }} to ${{ ctx.stage.name }}"
        description: "Deploying ${{ imageFrom(vars.imageRepo).RepoURL }}:${{ imageFrom(vars.imageRepo).Tag }} to ${{ ctx.stage.name }} environment. Promotion ID: ${{ ctx.promotion.name }}. Freight: ${{ ctx.targetFreight.name }}."
        issueType: Task
        assigneeEmail: devops@company.com
        labels:
          - deployment
          - "${{ ctx.stage.name }}"
          - "release-${{ imageFrom(vars.imageRepo).Tag }}"
  # Use the created issue key in subsequent steps
  - uses: jira
    config:
      credentials:
        secretName: jira-credentials
      updateIssue:
        issueKey: "${{ outputs['create-deployment-issue'].key }}"
        status: "IN PROGRESS"
```

### Update Issue

Updates an existing Jira issue with new information.

#### Configuration

| Name                         | Type     | Required | Description                                                       |
| ---------------------------- | -------- | -------- | ----------------------------------------------------------------- |
| `updateIssue.issueKey`       | `string` | Y        | The Jira Issue Key (e.g., EXT-123).                               |
| `updateIssue.summary`        | `string` | N        | Updated summary or title of the issue.                            |
| `updateIssue.description`    | `string` | N        | Updated description. Supports markdown formatting.                |
| `updateIssue.adfDescription` | `object` | N        | ADF content for complex formatting. Alternative to `description`. |
| `updateIssue.issueType`      | `string` | N        | Updated issue type.                                               |
| `updateIssue.assigneeEmail`  | `string` | N        | Email of the user to assign the issue to.                         |
| `updateIssue.status`         | `string` | N        | Status to set for the issue (e.g., 'IN PROGRESS', 'DONE').        |
| `updateIssue.addLabels`      | `array`  | N        | Labels to add to the issue.                                       |
| `updateIssue.removeLabels`   | `array`  | N        | Labels to remove from the issue.                                  |
| `updateIssue.customFields`   | `object` | N        | Custom fields to update.                                          |

#### Output

This step does not produce any output values.

#### Example

This example updates an existing issue's status and adds a comment with deployment details.

```yaml
steps:
  - uses: jira
    config:
      credentials:
        secretName: jira-credentials
      updateIssue:
        issueKey: DEPLOY-123
        status: "IN PROGRESS"
        summary: "Deploy ${{ imageFrom(vars.imageRepo).Tag }} to ${{ ctx.stage.name }} - IN PROGRESS"
        addLabels:
          - deploying
          - "${{ ctx.stage.name }}-deployment"
        customFields:
          customfield_10000: "${{ ctx.stage.name }} Environment"
          customfield_10001: "${{ ctx.promotion.name }}"
```

### Delete Issue

Deletes a Jira issue and optionally its subtasks.

#### Configuration

| Name                         | Type      | Required | Description                                    |
| ---------------------------- | --------- | -------- | ---------------------------------------------- |
| `deleteIssue.issueKey`       | `string`  | Y        | The Jira Issue Key (e.g., EXT-123).            |
| `deleteIssue.deleteSubtasks` | `boolean` | N        | If true, all subtasks will be deleted as well. |

#### Output

This step does not produce any output values.

#### Example

This example deletes a Jira issue and all its subtasks when a deployment is rolled back.

```yaml
steps:
  - uses: jira
    config:
      credentials:
        secretName: jira-credentials
      deleteIssue:
        issueKey: "${{ freightMetadata(ctx.targetFreight.name, 'jira-issue-key') }}"
        deleteSubtasks: true
```

### Search Issues

Searches for Jira issues using JQL (Jira Query Language).

#### Configuration

| Name                         | Type      | Required | Description                                                                                   |
| ---------------------------- | --------- | -------- | --------------------------------------------------------------------------------------------- |
| `searchIssue.jql`            | `string`  | Y        | The JQL query to search for issues.                                                           |
| `searchIssue.expectMultiple` | `boolean` | N        | If true, expects multiple results. If false, expects single result and fails with >1 results. |
| `searchIssue.fields`         | `array`   | N        | List of fields to include in search results.                                                  |
| `searchIssue.expands`        | `array`   | N        | List of fields to expand in search results.                                                   |

#### Output

| Name    | Type     | Description                                                                 |
| ------- | -------- | --------------------------------------------------------------------------- |
| `issue` | `object` | The found Jira issue object containing all requested fields and expansions. |

#### Example

This example searches for open deployment issues in a specific project and expects multiple results.

```yaml
steps:
  - uses: jira
    as: find-open-deployments
    config:
      credentials:
        secretName: jira-credentials
      searchIssue:
        jql: 'project = DEPLOY AND status != "Done" AND labels = "${{ ctx.stage.name }}-deployment" AND created >= -7d'
        expectMultiple: true
        fields:
          - summary
          - status
          - assignee
          - created
        expands:
          - changelog
  # Use search results in subsequent steps to notify team
# Note: This is just an example of using search outputs and may not be syntactically valid
- uses: http
  config:
    method: POST
    url: https://slack.com/api/chat.postMessage
    headers:
    - name: Authorization
      value: "Bearer ${{ secret('slack-credentials').token }}"
    - name: Content-Type
      value: application/json
    body: |
      ${{ quote({
        "channel": "#deployments",
        "text": "Found " + string(len(outputs['find-open-deployments'].issue)) + " open deployment issues for " + ctx.stage.name + " environment"
      }) }}
```

## Comment Management

### Add Comment

Adds a comment to an existing Jira issue.

#### Configuration

| Name                      | Type     | Required | Description                                                |
| ------------------------- | -------- | -------- | ---------------------------------------------------------- |
| `commentOnIssue.issueKey` | `string` | Y        | The Jira Issue Key (e.g., EXT-123).                        |
| `commentOnIssue.body`     | `string` | N        | Text content of the comment.                               |
| `commentOnIssue.adfBody`  | `object` | N        | ADF content for complex formatting. Alternative to `body`. |

#### Output

| Name        | Type     | Description                                                          |
| ----------- | -------- | -------------------------------------------------------------------- |
| `commentID` | `string` | The ID of the created comment that can be used for later operations. |

#### Example

This example adds a comment to a Jira issue with deployment progress information.

```yaml
steps:
  - uses: jira
    as: add-progress-comment
    config:
      credentials:
        secretName: jira-credentials
      commentOnIssue:
        issueKey: "${{ freightMetadata(ctx.targetFreight.name, 'jira-issue-key') }}"
        body: "Deployment started at ${{ ctx.promotion.creationTimestamp }}. Environment: ${{ ctx.stage.name }}. Image: ${{ imageFrom(vars.imageRepo).RepoURL }}:${{ imageFrom(vars.imageRepo).Tag }}. Promotion: ${{ ctx.promotion.name }}. Status: Deploying to ${{ ctx.stage.name }} environment..."
  # Later use the comment ID if needed
  - uses: jira
    config:
      credentials:
        secretName: jira-credentials
      deleteComment:
        issueKey: "${{ freightMetadata(ctx.targetFreight.name, 'jira-issue-key') }}"
        commentID: "${{ outputs['add-progress-comment'].commentID }}"
```

### Delete Comment

Removes a specific comment from a Jira issue.

#### Configuration

| Name                      | Type     | Required | Description                         |
| ------------------------- | -------- | -------- | ----------------------------------- |
| `deleteComment.issueKey`  | `string` | Y        | The Jira Issue Key (e.g., EXT-123). |
| `deleteComment.commentID` | `string` | Y        | The ID of the comment to delete.    |

#### Output

This step does not produce any output values.

#### Example

This example deletes a specific comment from a Jira issue.

```yaml
steps:
  - uses: jira
    config:
      credentials:
        secretName: jira-credentials
      deleteComment:
        issueKey: "${{ freightMetadata(ctx.targetFreight.name, 'jira-issue-key') }}"
        commentID: "${{ outputs['previous-comment-step'].commentID }}"
```

## Status Tracking

### Wait for Status

Waits for a Jira issue to reach a specific status before proceeding.

#### Configuration

| Name                           | Type     | Required | Description                                                    |
| ------------------------------ | -------- | -------- | -------------------------------------------------------------- |
| `waitForStatus.issueKey`       | `string` | Y        | The Jira Issue Key (e.g., EXT-123).                            |
| `waitForStatus.expectedStatus` | `string` | Y        | The expected status to wait for (e.g., 'IN PROGRESS', 'DONE'). |

#### Output

This step does not produce any output values.

#### Example

This example waits for a change request issue to be approved before proceeding with deployment.

```yaml
steps:
  - uses: jira
    config:
      credentials:
        secretName: jira-credentials
      waitForStatus:
        issueKey: "${{ freightMetadata(ctx.targetFreight.name, 'change-request-key') }}"
        expectedStatus: "Approved"
  # Deployment steps continue after approval...
  - uses: helm-template
    config:
      path: ./charts
      vars:
        imageTag: "${{ imageFrom(vars.imageRepo).Tag }}"
        environment: "${{ ctx.stage.name }}"
```

## Freight Metadata Integration

The Jira extension automatically stores created issue keys in the Freight metadata, allowing subsequent stages to reference the same issue. This enables tracking a single Jira issue across multiple promotion stages.

### Accessing Issue Keys from Freight Metadata

Use the `freightMetadata` template function to retrieve issue keys stored by previous stages:

```yaml
# Access the default issue key
issueKey: ${{ freightMetadata(ctx.targetFreight.name, 'jira-issue-key') }}

# Access a custom issue key (when issueAlias was used during creation)
issueKey: ${{ freightMetadata(ctx.targetFreight.name, 'my-custom-alias') }}
```

## Multi-Stage Workflow Example

This comprehensive example demonstrates how to use the Jira extension across multiple stages in a promotion pipeline, tracking a single issue from creation through production deployment:

```yaml
---
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: test
  namespace: kargo-proj
spec:
  requestedFreight:
    - origin:
        kind: Warehouse
        name: nginx
      sources:
        direct: true
  promotionTemplate:
    spec:
      vars:
        - name: imageRepo
          value: public.ecr.aws/nginx/nginx
      steps:
        # Create initial deployment ticket
        - as: create-deployment-ticket
          uses: jira
          config:
            credentials:
              secretName: jira
            createIssue:
              projectKey: KD
              issueType: Task
              summary: "Deploy Release ${{ imageFrom(vars.imageRepo).Tag }}"
              assigneeEmail: "devops@company.com"
              adfDescription:
                type: doc
                version: 1
                content:
                  - type: paragraph
                    content:
                      - type: text
                        text: " "
                  - type: heading
                    attrs:
                      level: 3
                    content:
                      - type: text
                        text: "Automated deployment ticket for release "
                      - type: text
                        text: "${{ imageFrom(vars.imageRepo).Tag }}"
                        marks:
                          - type: code
                  - type: paragraph
                    content:
                      - type: text
                        text: "Image:"
                        marks:
                          - type: strong
                      - type: text
                        text: " "
                      - type: text
                        text: "${{ imageFrom(vars.imageRepo).RepoURL }}:${{ imageFrom(vars.imageRepo).Tag }}"
                        marks:
                          - type: code
                  - type: paragraph
                    content:
                      - type: text
                        text: "Project:"
                        marks:
                          - type: strong
                      - type: text
                        text: " "
                      - type: text
                        text: "${{ ctx.project }}"
                        marks:
                          - type: code
              labels:
                - "automated-deployment"
                - "env-${{ ctx.stage.name }}"
                - "release-${{ imageFrom(vars.imageRepo).Tag }}"
                - "project-${{ ctx.project }}"

        # Update application
        - as: update-app
          uses: argocd-update
          config:
            apps:
              - name: test-app
                namespace: argocd
                sources:
                  - repoURL: https://github.com/company/app-config.git
                    kustomize:
                      images:
                        - repoURL: public.ecr.aws/nginx/nginx
                          tag: ${{ imageFrom("public.ecr.aws/nginx/nginx").Tag }}

        # Add progress comment
        - as: comment-on-ticket
          uses: jira
          config:
            credentials:
              secretName: jira
            commentOnIssue:
              issueKey: ${{ outputs['create-deployment-ticket'].key }}
              body: "Release ${{ imageFrom(vars.imageRepo).Tag }} has been promoted to ${{ ctx.stage.name }} environment at ${{ ctx.promotion.creationTimestamp }}. Freight: ${{ ctx.targetFreight.name }}. Ready for testing."

        # Cleanup on failure
        - as: on-failure-cleanup-issue
          uses: jira
          if: ${{ failure() }}
          config:
            credentials:
              secretName: jira
            deleteIssue:
              issueKey: ${{ outputs['create-deployment-ticket'].key }}
              deleteSubtasks: true

---
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: stage
  namespace: kargo-proj
spec:
  requestedFreight:
    - origin:
        kind: Warehouse
        name: nginx
      sources:
        stages:
          - test
  promotionTemplate:
    spec:
      vars:
        - name: imageRepo
          value: public.ecr.aws/nginx/nginx
      steps:
        # Wait for manual approval to proceed to staging
        - as: wait-approval
          uses: jira
          config:
            credentials:
              secretName: jira
            waitForStatus:
              issueKey: ${{ freightMetadata(ctx.targetFreight.name, 'jira-issue-key') }}
              expectedStatus: STAGED

        # Update staging application
        - as: update-app
          uses: argocd-update
          config:
            apps:
              - name: stage-app
                namespace: argocd
                sources:
                  - repoURL: https://github.com/company/app-config.git
                    kustomize:
                      images:
                        - repoURL: ${{ vars.imageRepo }}
                          tag: ${{ imageFrom(vars.imageRepo).Tag }}

        # Update ticket with staging progress
        - as: comment-on-ticket
          uses: jira
          config:
            credentials:
              secretName: jira
            commentOnIssue:
              issueKey: ${{ freightMetadata(ctx.targetFreight.name, 'jira-issue-key') }}
              body: "Release ${{ imageFrom(vars.imageRepo).Tag }} has been promoted to ${{ ctx.stage.name }} environment at ${{ ctx.promotion.creationTimestamp }}. Promotion: ${{ ctx.promotion.name }}. Status: Deployed and ready for staging validation."

        # Update environment labels
        - as: update-ticket-labels
          uses: jira
          config:
            credentials:
              secretName: jira
            updateIssue:
              issueKey: ${{ freightMetadata(ctx.targetFreight.name, 'jira-issue-key') }}
              removeLabels:
                - "env-test"
              addLabels:
                - "env-${{ ctx.stage.name }}"
                - "promotion-${{ ctx.promotion.name }}"

        # Cleanup comments on failure
        - as: on-failure-cleanup-comment
          uses: jira
          if: ${{ failure() && status('comment-on-ticket') == 'Succeeded' }}
          config:
            credentials:
              secretName: jira
            deleteComment:
              issueKey: ${{ freightMetadata(ctx.targetFreight.name, 'jira-issue-key') }}
              commentID: ${{ quote(outputs['comment-on-ticket'].commentID) }}

---
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: prod
  namespace: kargo-proj
spec:
  requestedFreight:
    - origin:
        kind: Warehouse
        name: nginx
      sources:
        stages:
          - stage
  promotionTemplate:
    spec:
      vars:
        - name: imageRepo
          value: public.ecr.aws/nginx/nginx
      steps:
        # Find the issue by searching for release label
        - as: search-issue
          uses: jira
          config:
            credentials:
              secretName: jira
            searchIssue:
              jql: "created <= 1d and labels IN (release-${{ imageFrom(vars.imageRepo).Tag }}) ORDER BY created DESC"
              expectMultiple: true
              fields:
                - key

        # Wait for final production approval
        - as: wait-approval
          uses: jira
          config:
            credentials:
              secretName: jira
            waitForStatus:
              issueKey: ${{ outputs['search-issue'].key }}
              expectedStatus: RELEASED

        # Deploy to production
        - as: update-app
          uses: argocd-update
          config:
            apps:
              - name: prod-app
                namespace: argocd
                sources:
                  - repoURL: https://github.com/company/app-config.git
                    kustomize:
                      images:
                        - repoURL: public.ecr.aws/nginx/nginx
                          tag: ${{ imageFrom("public.ecr.aws/nginx/nginx").Tag }}

        # Add final completion comment
        - as: comment-on-ticket
          uses: jira
          config:
            credentials:
              secretName: jira
            commentOnIssue:
              issueKey: ${{ outputs['search-issue'].key }}
              body: "Release ${{ imageFrom(vars.imageRepo).Tag }} has been successfully promoted to ${{ ctx.stage.name }} environment at ${{ ctx.promotion.creationTimestamp }}. Deployment completed for promotion ${{ ctx.promotion.name }}. All systems operational and release is live!"

        # Update to production labels
        - as: update-ticket-labels
          uses: jira
          config:
            credentials:
              secretName: jira
            updateIssue:
              issueKey: ${{ outputs['search-issue'].key }}
              removeLabels:
                - "env-stage"
              addLabels:
                - "env-${{ ctx.stage.name }}"
                - "released-${{ imageFrom(vars.imageRepo).Tag }}"
                - "promotion-${{ ctx.promotion.name }}"

        # Cleanup on failure
        - as: on-failure-cleanup-comment
          uses: jira
          if: ${{ failure() && status('comment-on-ticket') == 'Succeeded' }}
          config:
            credentials:
              secretName: jira
            deleteComment:
              issueKey: ${{ outputs['search-issue'].key }}
              commentID: ${{ quote(outputs['comment-on-ticket'].commentID) }}
```

This multi-stage workflow demonstrates:

- **Issue Creation**: The `test` stage creates a comprehensive Jira issue with ADF formatting
- **Freight Metadata**: The issue key is automatically stored in freight metadata for later stages
- **Status Tracking**: Each stage waits for specific approval statuses before proceeding
- **Progressive Updates**: Labels and comments are updated as the release moves through environments
- **Error Handling**: Cleanup steps run on failures to maintain clean state
- **Search Functionality**: The `prod` stage demonstrates finding issues by label when freight metadata isn't available
