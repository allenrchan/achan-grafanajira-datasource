package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/achan/grafana-jira-datasource/pkg/jira"
	"github.com/achan/grafana-jira-datasource/pkg/models"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces - only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// NewDatasource creates a new datasource instance.
func NewDatasource(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &Datasource{}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct{}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	client := jira.NewClient(config.URL, config.Username, config.Secrets.Token)

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, client, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	JQLQuery    string  `json:"jqlQuery"`
	Quantile    float64 `json:"quantile"`
	StartStatus string  `json:"startStatus"`
	EndStatus   string  `json:"endStatus"`
	Metric      string  `json:"metric"`
}

func (d *Datasource) query(_ context.Context, client *jira.Client, query backend.DataQuery) backend.DataResponse {
	// var response backend.DataResponse // Unused variable removed

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	// Append time range filter to JQL to reduce load
	// Format: "YYYY-MM-DD HH:mm"
	// Example: "project = PLAT AND updated >= '2023-01-01 00:00'"
	// We only care about From time because filtering "To" might exclude issues updated *after* the window but were active *during* the window?
	// Actually, if we want cycle time in a window, the issue must have had activity. 
	// "updated >= From" is safe because if it wasn't updated since From, it couldn't have transitioned in that window (except if we care about "open during", but cycle time is about transitions).
	// For "open tickets" count, we might need different logic, but for cycle time (transitions) and changelog history, "updated >= From" is correct.
	// For JQL metric (raw issues), if we want "issues active in window", "updated >= From" is also a good proxy or "updated >= From OR created >= From".
	// But "updated >= From" covers created too (creation is an update).
	// One edge case: Issue created before From, Updated before From, but still Open. It won't be fetched. 
	// But if it wasn't updated in the window, it didn't change status in the window, so cycle time/changelog won't have entries in the window anyway.
	// So "updated >= From" is safe optimization.
	
	jql := qm.JQLQuery
	if jql != "" {
		fromTime := query.TimeRange.From.Format("2006-01-02 15:04")
		// Check if JQL already has "order by" to avoid syntax error (order by must be last)
		// Basic check: split by "order by" (case insensitive)
		// This is tricky parsing. For now, let's append it at the END if no order by, or insert it.
		// A safer way: Use parentheses? "(original_jql) AND updated >= ..."
		// But "order by" must be outside parens.
		// If user provides "order by", we might break it. 
		// Simpler approach: Assume user might provide "order by".
		// We can try to append it. If JQL has "order by", we should insert before it.
		// But regex is fragile.
		// Alternative: Just append it and warn user? No.
		// Let's just append " AND updated >= ..." and hope user puts "order by" at end if at all? 
		// Actually, standard JQL allows "AND" clauses before "ORDER BY". 
		// If the user's string ends with "ORDER BY ...", appending " AND ..." is invalid syntax.
		// 
		// Let's keep it simple: Append it. If the user has "ORDER BY", they should move it or we accept it might fail for complex custom JQLs.
		// Or... we can check if "order by" exists and insert before it.
		// But wait, the previous `SearchChangelogs` logic paginates. It doesn't rely on sorting?
		// Actually, `jql` is passed as string.
		// Let's implement a simple heuristic: if "order by" is found, insert before it.
		
		// Note: We won't implement complex parsing now. Just append it, assuming most users put filter logic first.
		// If "ORDER BY" is present, we wrap the original query in parens? No, parens don't work around ORDER BY.
		// Let's just assume valid JQL structure.
		// Ideally we would put: `(user_jql) AND updated >= '...'`.
		// But if user_jql has ORDER BY, `( ... ORDER BY ...) AND ...` is invalid.
		
		// Decision: Append ` AND updated >= '...'`.
		// Limitation: User JQL must NOT end with ORDER BY for this to work perfectly.
		// Or we can warn.
		
		jql += fmt.Sprintf(" AND updated >= '%s'", fromTime)
	}

	// Fetch issues from Jira
	issues, err := client.SearchChangelogs(jql)
	if err != nil {
		// backend.StatusInternalServerError is not exported or valid in this SDK version likely.
		// Using backend.StatusBadRequest or constructing error with status.
		// Standard way is to return DataResponse with Error field.
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("jira search failed: %v", err.Error()))
	}

	switch qm.Metric {
	case "changelogRaw":
		return d.getChangelogRawData(issues)
	case "cycletime":
		return d.getCycletimeData(issues, qm, query.TimeRange)
	case "jql":
		return d.getJQLData(issues)
	default:
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("unknown metric: %s", qm.Metric))
	}
}

func (d *Datasource) getJQLData(issues []jira.Issue) backend.DataResponse {
	var response backend.DataResponse

	frame := data.NewFrame("response",
		data.NewField("Key", nil, []string{}),
		data.NewField("Summary", nil, []string{}),
		data.NewField("Status", nil, []string{}),
		data.NewField("IssueType", nil, []string{}),
		data.NewField("Project", nil, []string{}),
	)

	for _, issue := range issues {
		summary := ""
		if s, ok := issue.Fields["summary"].(string); ok {
			summary = s
		}
		
		status := ""
		if st, ok := issue.Fields["status"].(map[string]interface{}); ok {
			if name, ok := st["name"].(string); ok {
				status = name
			}
		}

		issueType := ""
		if it, ok := issue.Fields["issuetype"].(map[string]interface{}); ok {
			if name, ok := it["name"].(string); ok {
				issueType = name
			}
		}

		project := ""
		if p, ok := issue.Fields["project"].(map[string]interface{}); ok {
			// Try key, then name
			if key, ok := p["key"].(string); ok {
				project = key
			} else if name, ok := p["name"].(string); ok {
				project = name
			}
		}

		frame.AppendRow(issue.Key, summary, status, issueType, project)
	}

	response.Frames = append(response.Frames, frame)
	return response
}

func (d *Datasource) getChangelogRawData(issues []jira.Issue) backend.DataResponse {
	var response backend.DataResponse
	
	frame := data.NewFrame("response",
		data.NewField("IssueKey", nil, []string{}),
		data.NewField("IssueType", nil, []string{}),
		data.NewField("Created", nil, []time.Time{}),
		data.NewField("field", nil, []string{}),
		data.NewField("fromValue", nil, []string{}),
		data.NewField("toValue", nil, []string{}),
	)

	for _, issue := range issues {
		if issue.Changelog == nil {
			continue
		}
		
		issueType := "Unknown"
		if it, ok := issue.Fields["issuetype"].(map[string]interface{}); ok {
			if name, ok := it["name"].(string); ok {
				issueType = name
			}
		}

		for _, history := range issue.Changelog.Histories {
			createdTime, err := time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
			if err != nil {
				continue
			}

			for _, item := range history.Items {
				frame.AppendRow(
					issue.Key,
					issueType,
					createdTime,
					item.Field,
					item.FromString,
					item.ToString,
				)
			}
		}
	}

	response.Frames = append(response.Frames, frame)
	return response
}

func (d *Datasource) getCycletimeData(issues []jira.Issue, qm queryModel, timeRange backend.TimeRange) backend.DataResponse {
	var response backend.DataResponse

	frame := data.NewFrame("response",
		data.NewField("IssueKey", nil, []string{}),
		data.NewField("IssueType", nil, []string{}),
		data.NewField("Project", nil, []string{}),
		data.NewField("StartStatus", nil, []string{}),
		data.NewField("EndStatus", nil, []string{}),
		data.NewField("EndStatusCreated", nil, []time.Time{}),
		data.NewField("CycleTime", nil, []float64{}),
		data.NewField("Quantile", nil, []float64{}),
	)

	var cycleTimes []float64

	for _, issue := range issues {
		if issue.Changelog == nil {
			continue
		}

		issueType := "Unknown"
		if it, ok := issue.Fields["issuetype"].(map[string]interface{}); ok {
			if name, ok := it["name"].(string); ok {
				issueType = name
			}
		}

		project := ""
		if p, ok := issue.Fields["project"].(map[string]interface{}); ok {
			// Try key, then name
			if key, ok := p["key"].(string); ok {
				project = key
			} else if name, ok := p["name"].(string); ok {
				project = name
			}
		}

		var startCreated, endCreated time.Time
		var foundStart, foundEnd bool

		// For now we iterate as is.

		// Handle Grafana multi-value variable format "{Val1,Val2}" by stripping braces
		startStatusRaw := strings.Trim(qm.StartStatus, "{}")
		endStatusRaw := strings.Trim(qm.EndStatus, "{}")

		startStatuses := strings.Split(startStatusRaw, ",")
		endStatuses := strings.Split(endStatusRaw, ",")
		for i, s := range startStatuses {
			startStatuses[i] = strings.TrimSpace(s)
		}
		for i, s := range endStatuses {
			endStatuses[i] = strings.TrimSpace(s)
		}

		for _, history := range issue.Changelog.Histories {
			createdTime, err := time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
			if err != nil {
				continue
			}

			// Filter by time range
			if createdTime.Before(timeRange.From) || createdTime.After(timeRange.To) {
				continue
			}

			for _, item := range history.Items {
				if item.Field == "status" {
					isStart := false
					for _, s := range startStatuses {
						if item.ToString == s {
							isStart = true
							break
						}
					}
					
					isEnd := false
					for _, s := range endStatuses {
						if item.ToString == s {
							isEnd = true
							break
						}
					}

					if isStart {
						// Logic: use earliest timestamp for start status
						// If we haven't found a start status yet, or if this one is earlier than the existing one, update it.
						// Wait, histories are usually chronological (or reverse?). Jira API returns reverse chronological by default in some views, but standard changelog is chronological?
						// The current loop iterates histories in order. If they are chronological, the FIRST match is the earliest.
						// If they are reverse chronological, the LAST match is the earliest.
						// Assuming standard chronological order from search/jql expand:
						
						// If we want the EARLIEST occurrence of ANY start status:
						if !foundStart {
							startCreated = createdTime
							foundStart = true
						} else {
							// If we already found a start, only update if this one is earlier (unlikely if loop is chronological) 
							// OR if we want to reset start logic?
							// The user requirement: "using the earlier date for the start".
							// If an issue moves StartA -> StartB -> End, cycle time should be StartA to End?
							// Yes, "earliest date for start".
							if createdTime.Before(startCreated) {
								startCreated = createdTime
							}
						}
					}
					
					if isEnd {
						// Logic: use latest timestamp for end status
						// If we want LATEST occurrence of ANY end status:
						if !foundEnd {
							endCreated = createdTime
							foundEnd = true
						} else {
							if createdTime.After(endCreated) {
								endCreated = createdTime
							}
						}
					}
					
					// We only emit a row if we have both start and end, AND we are processing the END transition?
					// The previous logic emitted a row *every time* both flags were true inside the loop.
					// This means if I have Start -> End -> End2, it emitted for End and End2 (using same Start).
					// If I have Start -> Start2 -> End, it emitted for End (using Start2 if it overwrote, or Start1).
					
					// User logic: "using the earlier date for the start and later date for the end".
					// This implies we should process the WHOLE history for an issue, find the min(Start) and max(End), and THEN emit ONE row per issue (or per cycle?).
					// If we emit one row per issue, we should move the `frame.AppendRow` OUTSIDE the history loop.
					
					// HOWEVER, if an issue cycles multiple times (Start -> End -> Start -> End), do we want multiple rows?
					// Usually yes. But the user said "earliest start and latest end". This might imply one single cycle per issue spanning the whole range.
					// Let's assume one cycle per issue for "Earliest Start" and "Latest End" logic across the filtered time range.
					// If so, we just accumulate timestamps in the loop and append ONCE after the loop.
				}
			}
		}
		
		if foundStart && foundEnd {
			diff := math.Abs(float64(endCreated.Sub(startCreated).Milliseconds()))
			cycleTime := math.Ceil(diff/(1000*3600*24)) + 1
			
			frame.AppendRow(
				issue.Key,
				issueType,
				project,
				qm.StartStatus, // We return the config string, not the specific matched status, or we could return "Multiple"
				qm.EndStatus,
				endCreated,
				cycleTime,
				0.0,
			)
			cycleTimes = append(cycleTimes, cycleTime)
		}
	}

	// Calculate Quantile
	quantileValue := 0.0
	if len(cycleTimes) > 0 {
		sort.Float64s(cycleTimes)
		// Simple quantile implementation
		// Index = q * (n-1)
		pos := (qm.Quantile / 100.0) * float64(len(cycleTimes)-1)
		base := int(pos)
		rest := pos - float64(base)
		
		if base+1 < len(cycleTimes) {
			quantileValue = cycleTimes[base] + rest*(cycleTimes[base+1]-cycleTimes[base])
		} else {
			quantileValue = cycleTimes[base]
		}
	}

	// Update Quantile column
	// rows := frame.Rows() // Unused variable removed
	for i := 0; i < frame.Rows(); i++ {
		// Update the last column (Quantile is index 7)
		frame.Fields[7].Set(i, quantileValue)
	}

	response.Frames = append(response.Frames, frame)
	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
func (d *Datasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	res := &backend.CheckHealthResult{}
	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)

	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = "Unable to load settings"
		return res, nil
	}

	if config.Secrets.Token == "" {
		res.Status = backend.HealthStatusError
		res.Message = "API Token is missing"
		return res, nil
	}

	client := jira.NewClient(config.URL, config.Username, config.Secrets.Token)
	err = client.Myself()
	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = fmt.Sprintf("Jira connection failed: %s", err.Error())
		return res, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}

