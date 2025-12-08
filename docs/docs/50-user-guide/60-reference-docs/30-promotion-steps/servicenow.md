---
sidebar_label: servicenow
description: Integrates with ServiceNow to manage Change Requests, Incidents, Problems, etc., and track promotion workflows.
---

# `snow`

<span class="tag professional"></span>
<span class="tag beta"></span>

:::info
This promotion step is only available in Kargo on the
[Akuity Platform](https://akuity.io/akuity-platform), versions v1.9.0 and above.
:::

The `snow` promotion step provides comprehensive integration with ServiceNow, allowing you
to create, update, delete, and search for records, and track
promotion workflows. This is particularly useful for maintaining traceability between
your promotion processes and project management activities.

This promotion step supports various operations including managing Change Requests, Incidents, 
Problems, and other records in ServiceNow. It also provides status tracking, making it a 
powerful tool for promotion workflows that require coordination with project management systems.

## Credentials Configuration

All ServiceNow operations require proper authentication credentials stored in a Kubernetes
`Secret`.

| Name                     | Type     | Required | Description                                                                                     |
| ------------------------ | -------- | -------- | ----------------------------------------------------------------------------------------------- |
| `credentials.secretName` | `string` | Y        | Name of the `Secret` containing the ServiceNow credentials in the project namespace.            |
| `credentials.type`       | `string` | Y        | Type of ServiceNow credentials to use for authentication (either `api-token` or `basic`).       |

For `credentials.type: api-token` the referenced `Secret` should contain the following keys:

- `apiToken`: ServiceNow API Token (see [this blog post](https://www.servicenow.com/community/developer-advocate-blog/inbound-rest-api-keys/ba-p/2854924) for how to create an API token in ServiceNow).
- `instanceURL`: Your ServiceNow instance URL.

![](./images/snow-instance-url.png)

For `credentials.type: basic` the referenced `Secret` should contain the following keys:

- `username`: Username of the ServiceNow user (you may want to [create a user](https://www.servicenow.com/docs/bundle/zurich-platform-administration/page/administer/users-and-groups/task/t_CreateAUser.html) specifically for this integration).
- `password`: Password of the ServiceNow user (for how to set the password for a user, see [this](https://www.servicenow.com/docs/bundle/zurich-platform-security/page/integrate/authentication/task/reset-your-password.html)).
- `instanceURL`: Your ServiceNow instance URL.

## Using the API

Most of the time you cannot use the field labels you see in the ServiceNow UI as keys in the REST API. 
For example, if you want to set the value for the “Short description” field:

![](./images/snow-short-des.png)

You can't use `Short description` in the REST API. You need to use `short_description` as the key and set the value via REST API parameters.
To find the correct key for a field, right-click on the field and click `Configure Dictionary`:

![](./images/snow-configure-dict.png)

`column_name` is the key:

![](./images/snow-col-name.png)

## Record Management

### Create Record

`snow` integration supports two ServiceNow APIs:  
#### 1. Change Management API
This API is used primarily for managing Change Requests.

Official documentation is available [here](https://www.servicenow.com/docs/bundle/zurich-api-reference/page/integrate/inbound-rest/concept/change-management-api.html).

URL format: `/api/sn_chg_rest/{api_version}/change/{change_sys_id}/task/{task_sys_id}`

#### 2. Table API
This API is used for managing Incidents, Problems, and other record types.

Official documentation is available [here](https://www.servicenow.com/docs/bundle/zurich-api-reference/page/integrate/inbound-rest/concept/c_TableAPI.html).

URL format: `/api/now/{api_version}/table/{tableName}/{sys_id}`

#### Configuration

| Name                 | Type         | Required | Description                                                                                                                     |
| -------------------- | ------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `parameters`         | `string map` | Y        | Parameters/fields of the record.                                                                                                |
| `tableName`          | `string`     | Y        | Table name of the record type you want to create (e.g., `incident`, `problem`, `change_request`).                              |
| `template`           | `object`     | N        | Specify this and the subfields below to create a Change Request using the Change Management API (if omitted, the Table API is used). |
| `template.type`      | `string`     | Y/N      | Template type (`standard`, `emergency`, or `normal`). Required if `template` is specified; otherwise optional.                 |
| `template.templateId`| `string`     | Y/N      | Template ID of the standard template (required if `template.type` is `standard`; otherwise optional).                          |

#### Output

| Name     | Type     | Description                                                                 |
| -------- | -------- | ----------------------------------------------------------------------------- |
| `sys_id` | `string` | The `sys_id` of the created ServiceNow record (e.g., `ed89b72c83c172104517e470ceaad30a`). |
| `number` | `string` | The `number` of the created ServiceNow record (e.g., `CH-123`).             |

#### Example

This example creates a new Change Request using the Change Management API:

```yaml
steps:
  - as: snowcreate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        impact: "3"
        short_description: Deploy to ${{ ctx.stage }} complete for ${{ vars.imageRepo
          }}:${{ imageFrom(vars.imageRepo).Tag }}
        urgency: "3"
      tableName: change_request
      template:
        templateId: ed89b72c83c172104517e470ceaad30a # example
        type: standard
    uses: snow-create
# Use the created snow record number in subsequent steps
  - as: snowquery
    config:
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      query:
        number: ${{ task.outputs.snowcreate.number }}
      tableName: change_request
    uses: snow-query-records
```

This example creates a new Incident using the Table API:

```yaml
steps:
  - as: snowcreate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        impact: "3"
        short_description: Deploy to ${{ ctx.stage }} complete for ${{ vars.imageRepo }}:${{ imageFrom(vars.imageRepo).Tag }}
        urgency: "3"
      tableName: incident
    uses: snow-create
# Use the created snow record number in subsequent steps
  - as: snowquery
    config:
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      query:
        number: ${{ task.outputs.snowcreate.number }}
      tableName: incident
    uses: snow-query-records
```

This example creates a new Change Request using the Table API (notice how the `template` field is omitted):

```yaml
steps:
  - as: snowcreate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        impact: "3"
        short_description: Deploy to ${{ ctx.stage }} complete for ${{ vars.imageRepo
          }}:${{ imageFrom(vars.imageRepo).Tag }}
        urgency: "3"
      tableName: change_request
    uses: snow-create
# Use the created snow record number in subsequent steps
  - as: snowquery
    config:
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      query:
        number: ${{ task.outputs.snowcreate.number }}
      tableName: change_request
    uses: snow-query-records
```

### Update Record

Updates an existing ServiceNow record with new information.

#### Configuration

| Name            | Type         | Required | Description                                                                                                                     |
| --------------- | ------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `parameters`    | `string map` | Y        | Parameters/fields of the record.                                                                                                |
| `tableName`     | `string`     | Y        | Table name of the record type you want to update (e.g., `incident`, `problem`, `change_request`).                              |
| `ticketId`      | `string`     | Y        | Ticket ID (`sys_id`) of the record you want to update.                                                                          |
| `template`      | `object`     | N        | Specify this to update a Change Request using the Change Management API (if omitted, the Table API is used).                    |
| `template.type` | `string`     | Y/N      | Template type (`standard`, `emergency`, or `normal`). Required if `template` is specified; otherwise optional.                 |

#### Output

This step does not produce any output.

#### Example

This example updates a Change Request using the Change Management API:

```yaml
steps:
  - as: snowupdate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        short_description: Update deployment for ${{ vars.imageRepo }}:${{
          imageFrom(vars.imageRepo).Tag }} to done
      tableName: change_request
      template:
        type: standard
      ticketId: 9d41c061c611228700edc88b231ec47c
    uses: snow-update
```

This example updates an Incident using the Table API:

```yaml
steps:
  - config:
      ticketId: ${{ task.outputs.snowquery.sys_id }}
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        short_description: Update deployment incident for ${{ vars.imageRepo }}:${{ imageFrom(vars.imageRepo).Tag }} to resolved
      tableName: incident
    uses: snow-update
```

This example updates a Change Request using the Table API (notice how the `template` field is omitted):

```yaml
steps:
  - as: snowupdate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        short_description: Update deployment incident for ${{ vars.imageRepo }}:${{
          imageFrom(vars.imageRepo).Tag }} to resolved
      tableName: change_request
      ticketId: ${{ task.outputs.snowquery.sys_id }}
    uses: snow-update
```

### Delete Record

Deletes a ServiceNow record.

#### Configuration

| Name            | Type     | Required | Description                                                                                                                     |
| --------------- | -------- | -------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `tableName`     | `string` | Y        | Table name of the record type you want to delete (e.g., `incident`, `problem`, `change_request`).                               |
| `ticketId`      | `string` | Y        | Ticket ID (`sys_id`) of the record you want to delete.                                                                          |
| `template`      | `object` | N        | Specify this to delete a Change Request using the Change Management API (if omitted, the Table API is used).                    |
| `template.type` | `string` | Y/N      | Template type (`standard`, `emergency`, or `normal`). Required if `template` is specified; otherwise optional.                 |

#### Output

This step does not produce any output.

#### Example

This example deletes a Change Request using the Change Management API:

```yaml
steps:
  - as: snowdelete
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      tableName: change_request
      template:
        type: standard
      ticketId: d457fbac6112287007379b57c6b2e60 # example
    uses: snow-delete
```

This example deletes an Incident using the Table API:

```yaml
steps:
  - as: snowdelete
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      tableName: incident
      ticketId: d457fbac6112287007379b57c6b2e60 # example
    uses: snow-delete
```

This example deletes a Change Request using the Table API (notice how the `template` field is omitted):

```yaml
steps:
  - as: snowdelete
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      tableName: change_request
      ticketId: d457fbac6112287007379b57c6b2e60 # example
    uses: snow-delete
```

### Query Records

Query for records and return the first matching record.

#### Configuration

| Name        | Type     | Required | Description                                                                                                                     |
| ----------- | -------- | -------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `tableName` | `string` | Y        | Table name of the record type you want to query (e.g., `incident`, `problem`, `change_request`).                                |
| `query`     | `object` | Y        | Specify your query parameters on which you want to filter the records. For supported parameters, please see the official documentation [here](https://www.servicenow.com/docs/bundle/zurich-api-reference/page/integrate/inbound-rest/concept/c_TableAPI.html#title_table-GET). |

#### Output

The output is a ServiceNow record. All fields are available for use in subsequent steps.

| Name     | Type     | Description                                           |
| -------- | -------- | ----------------------------------------------------- |
| `record` | `object` | The found ServiceNow record object containing all fields. |

#### Example

This example searches for Change Requests with a specific record `number`:

```yaml
steps:
  - as: snowquery
    config:
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      query:
        number: CHG0000007
      tableName: change_request
    uses: snow-query-records
# Use the queried record in subsequent steps
  - as: snowupdate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        short_description: Update incident for ${{ vars.imageRepo }}:${{
          imageFrom(vars.imageRepo).Tag }} to resolved
      tableName: change_request
      template:
        type: standard
      ticketId: ${{ task.outputs.snowquery.sys_id }}
    uses: snow-update
```

### Wait for Condition

Waits for a ServiceNow record field to be set to a particular value before proceeding.

#### Configuration

| Name        | Type     | Required | Description                                                                                                                     |
| ----------- | -------- | -------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `tableName` | `string` | Y        | Table name of the record type you want to wait on (e.g., `incident`, `problem`, `change_request`).                              |
| `ticketId`  | `string` | Y        | Ticket ID (`sys_id`) of the record you want to monitor.                                                                         |
| `condition` | `string` | Y        | Condition to wait on before proceeding.                                                                                         |

#### Output

This step does not produce any output.

#### Example

This example waits for a Change Request to be Scheduled (`state=-2`) before proceeding with promotion:

```yaml
steps:
  - as: snow-wait-for-condition
    config:
      condition: state=-2
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      tableName: change_request
      ticketId: d457fbac6112287007379b57c6b2e60
    uses: snow-wait-for-condition
# promotion steps continue after approval...
  - uses: helm-template
    config:
      path: ./charts
      vars:
        imageTag: "${{ imageFrom(vars.imageRepo).Tag }}"
        environment: "${{ ctx.stage }}"
```

You can write more complex conditions as well. See the list of supported operators [here](https://www.servicenow.com/docs/bundle/zurich-platform-user-interface/page/use/common-ui-elements/reference/r_OpAvailableFiltersQueries.html).

## Different E2E Workflows

These examples demonstrate the different steps supported for ServiceNow integration.

### Change API Workflow

```yaml
  - as: snowcreate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        impact: "3"
        short_description: Deploy to ${{ ctx.stage }} complete for ${{ vars.imageRepo
          }}:${{ imageFrom(vars.imageRepo).Tag }}
        urgency: "3"
      tableName: change_request
      template:
        templateId: ed89b72c83c172104517e470ceaad30a
        type: standard
    uses: snow-create
  - as: snowquery
    config:
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      query:
        number: ${{ task.outputs.snowcreate.number }}
      tableName: change_request
    uses: snow-query-records
  - as: snowupdate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        short_description: Update deployment for ${{ vars.imageRepo }}:${{
          imageFrom(vars.imageRepo).Tag }} to resolved
      tableName: change_request
      template:
        type: standard
      ticketId: ${{ task.outputs.snowquery.sys_id }}
    uses: snow-update
  - as: snow-wait-for-condition
    config:
      condition: state=-2
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      tableName: change_request
      ticketId: ${{ task.outputs.snowquery.sys_id }}
    uses: snow-wait-for-condition
  - as: snowdelete
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      tableName: change_request
      template:
        type: standard
      ticketId: ${{ task.outputs.snowquery.sys_id }}
    uses: snow-delete
```

### Table API Workflow

```yaml
  - as: snowcreate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        impact: "3"
        short_description: Deploy to ${{ ctx.stage }} complete for ${{ vars.imageRepo }}:${{ imageFrom(vars.imageRepo).Tag }}
        urgency: "3"
      tableName: incident
    uses: snow-create
  - as: snowquery
    config:
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      query:
        number: ${{ task.outputs.snowcreate.number }}
      tableName: incident
    uses: snow-query-records
  - config:
      ticketId: ${{ task.outputs.snowquery.sys_id }}
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        short_description: Update deployment for ${{ vars.imageRepo }}:${{ imageFrom(vars.imageRepo).Tag }} to resolved
      tableName: incident
    uses: snow-update
  - as: snow-wait-for-condition
    config:
      condition: state=6
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      tableName: incident
      ticketId: ${{ task.outputs.snowquery.sys_id }}
    uses: snow-wait-for-condition
  - as: snowdelete
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      tableName: incident
      ticketId: ${{ task.outputs.snowquery.sys_id }}
    uses: snow-delete
```

Here's a Table API workflow with Change Request:

```yaml
  - as: snowcreate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        impact: "3"
        short_description: Deploy to ${{ ctx.stage }} complete for ${{ vars.imageRepo
          }}:${{ imageFrom(vars.imageRepo).Tag }}
        urgency: "3"
      tableName: change_request
    uses: snow-create
  - as: snowquery
    config:
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      query:
        number: ${{ task.outputs.snowcreate.number }}
      tableName: change_request
    uses: snow-query-records
  - as: snowupdate
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      parameters:
        short_description: Update deployment incident for ${{ vars.imageRepo }}:${{
          imageFrom(vars.imageRepo).Tag }} to resolved
      tableName: change_request
      ticketId: ${{ task.outputs.snowquery.sys_id }}
    uses: snow-update
  - as: snow-wait-for-condition
    config:
      condition: state=-2
      credentials:
        namespace: kargo-demo
        secretName: snow-creds
        type: api-token
      tableName: change_request
      ticketId: ${{ task.outputs.snowquery.sys_id }}
    uses: snow-wait-for-condition
  - as: snowdelete
    config:
      credentials:
        secretName: snow-creds
        type: api-token
      tableName: change_request
      ticketId: ${{ task.outputs.snowquery.sys_id }}
    uses: snow-delete
```