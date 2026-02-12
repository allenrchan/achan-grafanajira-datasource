# Grafana Jira Datasource

This is a Grafana datasource plugin that connects to Jira Cloud to visualize issue data and calculate cycle time metrics.

## Features

*   **JQL (Raw Issue Data)**: Retrieve raw issue fields (Key, Summary, Status, Issue Type, Project) based on a JQL query. Supports full pagination to fetch all matching issues.
*   **Cycle Time**: Calculate the time it takes for issues to move between specific statuses (e.g., "In Progress" to "Done").
    *   **Multi-Status Support**: Define multiple start or end statuses (comma-separated or via variables) to capture transitions more flexibly.
    *   **Earliest Start / Latest End**: Automatically uses the earliest "Start" transition and latest "End" transition for accurate cycle calculation.
    *   **Quantile Calculation**: Computes and returns the specified quantile (e.g., 85th percentile) for the cycle time dataset.
*   **Project Partitioning**: Includes a "Project" field to allow grouping/partitioning data by project in visualizations (e.g., separate series in a Scatter Plot).
*   **Secure Authentication**: Uses Basic Auth (Email + API Token) securely handled by the backend, compatible with Jira Cloud.
*   **Optimized Performance**: Automatically filters JQL queries by the dashboard time range (`AND updated >= ...`) to minimize API load and avoid fetching stale history.
*   **Template Variables**: Supports Grafana template variables in JQL and Status fields, including multi-value variables (e.g., `${status:csv}`).

## Configuration

1.  **Add Data Source**: In Grafana, go to **Configuration** > **Data Sources** > **Add data source**, and select "Grafana Jira Datasource".
2.  **Settings**:
    *   **URL**: Your Jira Cloud instance URL (e.g., `https://your-domain.atlassian.net`).
    *   **Email**: The email address of your Atlassian account.
    *   **API Token**: Create an API token at [id.atlassian.com](https://id.atlassian.com/manage-profile/security/api-tokens) and paste it here.
3.  **Save & Test**: Click "Save & Test" to verify the connection.

## Usage

### Query Editor

In a dashboard panel, select the Jira datasource and configure the query:

*   **Metric**: Choose the type of data to visualize.
    *   **JQL (Raw Issue Data)**: Returns a table of issues matching your JQL. Useful for `Table` visualizations.
    *   **cycle time**: Returns cycle time metrics, useful for `Scatter Plot` or `Bar Gauge` visualizations.
*   **JQL Query**: Enter your JQL query (e.g., `project IN ('PROJ', 'QA')`).
    *   *Note*: The plugin automatically appends `AND updated >= <dashboard_from_time>` to the query. You do not need to manually add time filters unless you want to restrict it further.
*   **Start Status** (Cycle Time only): The status(es) where the cycle begins (e.g., `In Progress` or `In Progress, Review`).
*   **End Status** (Cycle Time only): The status(es) where the cycle ends (e.g., `Done` or `Closed, Released`).
*   **Quantile** (Cycle Time only): The percentile to calculate (e.g., `85` for 85th percentile).

### Multi-Series Visualization
To create a Scatter Plot with different colors per project:
1.  Use a JQL that selects multiple projects.
2.  In the Panel Editor, go to **Transformations**.
3.  Add **"Partition by values"**.
4.  Select the **"Project"** field.
5.  This will split the data into separate series for each project.

### Template Variables

You can use dashboard variables in your query fields to make dashboards interactive:
*   **JQL**: `project IN (${project:singlequote})`
*   **Status**: `${StartStatus}` (Mult-value variables are supported)

## Development

### Prerequisites

*   Node.js (v22+)
*   pnpm
*   Go (v1.21+)
*   Mage

### Build

1.  **Frontend**:
    ```bash
    pnpm install
    pnpm run build
    ```
2.  **Backend**:
    ```bash
    # Build for current platform
    mage -v build:backend
    
    # Build for Linux (e.g. for Docker/Production)
    mage -v build:linux
    ```

### Run Locally

```bash
pnpm run server
```
This starts a local Grafana instance with the plugin installed.
